// via github.com/HenrikJoreteg/favicon-setter
// modified to parameterize document

module.exports = function setFavicon(document, href) {
  var head = document.getElementsByTagName('head')[0]
  var faviconId = 'favicon'
  var link = document.createElement('link')
  var oldLink = document.getElementById(faviconId)
  link.id = faviconId
  link.rel = 'shortcut icon'
  link.href = href
  if (oldLink) {
    head.removeChild(oldLink)
  }
  head.appendChild(link)
}
