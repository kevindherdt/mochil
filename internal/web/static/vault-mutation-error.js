// Only the explicit server contract proves that retrying would repeat a write.
export function vaultMutationCommitted(error) {
  return error?.status === 500 &&
    error.code === 'vault_index_refresh_failed' &&
    error.committed === true && error.indexRefreshed === false;
}
