const GIB = 1073741824

export function managedIngestConfig(incomingValue, targetValue, maximumGiBValue) {
  const incoming = String(incomingValue || '').trim()
  const target = String(targetValue || '').trim()
  if (!incoming && !target) return null
  return {
    incoming_directory: incoming,
    managed_directory: target,
    maximum_file_bytes: Math.round(Number(maximumGiBValue) * GIB),
  }
}
