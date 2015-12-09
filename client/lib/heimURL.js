export default function heimURL(href) {
  let url = (process.env.HEIM_PREFIX || '') + href
  if (process.env.HEIM_GIT_COMMIT && /^\/static\//.test(href)) {
    url += '?v=' + process.env.HEIM_GIT_COMMIT
  }
  return url
}
