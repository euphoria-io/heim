// via github.com/HenrikJoreteg/favicon-setter
// modified to parameterize document

module.exports = function setFavicon(document, href) {
  var head = document.getElementsByTagName('head')[0]
  var faviconId = 'favicon'
  var oldLink = document.getElementById(faviconId)
  if (oldLink.getAttribute('href') == href) {
    return
  }
  var link = document.createElement('link')
  link.id = faviconId
  link.rel = 'shortcut icon'
  link.href = href
  if (oldLink) {
    head.removeChild(oldLink)
  }
  head.appendChild(link)
}
