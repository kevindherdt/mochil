export function createProgressSync({ save, resolveConflict, onConflict = () => {} }) {
  let revision = 0, generation = 0, conflicted = false, pending = Promise.resolve()

  function reset(value = 0) {
    generation++
    revision = Number(value) || 0
    conflicted = false
  }

  function submit(payload) {
    if (conflicted) return pending
    const currentGeneration = generation
    const result = pending.then(async () => {
      if (conflicted || currentGeneration !== generation) return false
      try {
        const saved = await save({ ...payload, expected_revision: revision })
        if (currentGeneration === generation) revision = Number(saved?.revision) || revision
        return currentGeneration === generation
      } catch (error) {
        if (currentGeneration !== generation) return false
        if (error?.status === 409 || String(error?.message).includes('changed on another device')) {
          if (resolveConflict) {
            const resolution = await resolveConflict(payload, error, () => currentGeneration === generation)
            if (currentGeneration !== generation) return false
            const remoteRevision = Number(resolution?.revision)
            if (Number.isFinite(remoteRevision)) revision = remoteRevision
            if (resolution?.retry === true) {
              const saved = await save({ ...payload, expected_revision: revision })
              if (currentGeneration === generation) revision = Number(saved?.revision) || revision
              return currentGeneration === generation
            }
            if (Number.isFinite(remoteRevision) || resolution?.handled === true) return false
          }
          conflicted = true
          onConflict(error)
          return false
        }
        throw error
      }
    })
    // Keep the serialization tail usable after transport/reconciliation errors.
    // The caller still receives the original rejection, but a later write must
    // be allowed to retry rather than inheriting that rejected promise forever.
    pending = result.catch(() => false)
    return result
  }

  return { reset, submit, get conflicted() { return conflicted }, get revision() { return revision } }
}
