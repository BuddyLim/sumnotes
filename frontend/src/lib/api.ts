const orignalFetch = window.fetch

let refreshPromise: Promise<Response> | null = null

window.fetch = async (...args) => {
  const [url, config] = args

  let response = await orignalFetch(url, config)

  if (response.status === 404) {
    window.location.href = 'http://localhost:9999/api/auth/google'
    return new Promise(() => {})
  }

  if (response.status === 401) {
    if (!refreshPromise) {
      refreshPromise = orignalFetch('http://localhost:9999/api/auth/refresh', {
        credentials: 'include',
      })
    }

    try {
      const refreshResponse = await refreshPromise

      if (!refreshResponse.ok) {
        throw new Error('Failed to refresh session')
      }

      response = await orignalFetch(url, config)
    } catch {
      window.location.href = 'http://localhost:9999/api/auth/google'
      return new Promise(() => {})
    } finally {
      refreshPromise = null
    }
  }

  return response
}
