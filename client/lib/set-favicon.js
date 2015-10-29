// via github.com/HenrikJoreteg/favicon-setter
// modified to parameterize document

export default function setFavicon(document, href) {
  const head = document.getElementsByTagName('head')[0]
  const faviconId = 'favicon'
  const oldLink = document.getElementById(faviconId)
  if (oldLink.getAttribute('href') === href) {
    return
  }
  const link = document.createElement('link')
  link.id = faviconId
  link.rel = 'shortcut icon'
  link.href = href
  if (oldLink) {
    head.removeChild(oldLink)
  }
  head.appendChild(link)
}
