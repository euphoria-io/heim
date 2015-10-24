module.exports = function(href) {
  return (process.env.HEIM_PREFIX || '') + href
}
