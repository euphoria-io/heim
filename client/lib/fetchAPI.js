export function postAPI(url, data) {
  return fetch(url, {
    method: 'post',
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  })
    .then(response => {
      // via https://github.com/github/fetch
      if (response.status >= 200 && response.status < 300) {
        return response
      }
      const error = new Error('request failed: ' + response.statusText)
      error.action = url
      error.response = response
      throw error
    })
    .then(response => response.json())
}
