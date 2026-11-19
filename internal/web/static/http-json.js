const maximumErrorDetailLength = 240

function compactText(value) {
  return String(value || '').replace(/\s+/g, ' ').trim().slice(0, maximumErrorDetailLength)
}

function responseError(response, body, data) {
  const serverMessage = compactText(data?.error || data?.message)
  if (serverMessage) return serverMessage

  const contentType = (response.headers.get('content-type') || '').toLowerCase()
  if (response.redirected || contentType.includes('text/html') || /^\s*<!doctype html|^\s*<html/i.test(body)) {
    return 'Mochil received a sign-in or web page instead of API data. Reload this page and sign in again, then retry.'
  }
	if (response.status === 401) {
		return 'Your Mochil web session expired. Reload this page and sign in again, then retry.'
	}

  const detail = compactText(body)
  return detail || `Mochil could not complete the request (HTTP ${response.status}).`
}

export async function requestJSON(url, options) {
  const response = await fetch(url, { priority: 'high', ...options })
  const body = await response.text()
  let data = null

  if (body.trim()) {
    try {
      data = JSON.parse(body)
    } catch {
      const error = new Error(responseError(response, body))
      error.status = response.status
      throw error
    }
  }

  if (!response.ok) {
    const error = new Error(responseError(response, body, data))
    error.status = response.status
    error.code = compactText(data?.code)
    error.committed = data?.committed === true
    error.indexRefreshed = data?.index_refreshed
    throw error
  }
  return { data, response }
}
