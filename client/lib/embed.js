import queryString from 'querystring'


const allowedImageDomains = {
  'imgs.xkcd.com': true,
  'i.ytimg.com': true,
}

let frozen = true
let _freezeHandler = null
window.addEventListener('message', function onMessage(ev) {
  if (ev.origin === process.env.HEIM_ORIGIN) {
    if (ev.data.type === 'freeze') {
      frozen = true
    } else if (ev.data.type === 'unfreeze') {
      frozen = false
    }
    if (_freezeHandler) {
      _freezeHandler(frozen)
    }
  }
}, false)

function handleFreeze(callback) {
  _freezeHandler = callback
  callback(frozen)
}

function loadImage(id, url, onloadCallback) {
  const img = document.createElement('img')
  img.src = url

  let widthSent = false

  function sendImageSize() {
    if (widthSent) {
      return
    }
    widthSent = true

    const displayHeight = document.body.offsetHeight
    let displayWidth
    const ratio = img.naturalWidth / img.naturalHeight
    if (ratio < 9 / 16) {
      displayWidth = 9 / 16 * displayHeight
      img.style.width = displayWidth + 'px'
      img.style.height = 'auto'
    } else {
      displayWidth = img.naturalWidth * (displayHeight / img.naturalHeight)
    }

    window.top.postMessage({
      id: id,
      type: 'size',
      data: {
        width: displayWidth,
      },
    }, process.env.HEIM_ORIGIN)
  }

  function checkImage() {
    if (img.naturalWidth) {
      sendImageSize()
    } else if (!widthSent) {
      setTimeout(checkImage, 100)
    }
  }

  img.onload = function onload() {
    sendImageSize()
    onloadCallback(img)
  }
  document.body.appendChild(img)
  checkImage()
}

function render() {
  const data = queryString.parse(location.search.substr(1))

  if (data.kind === 'imgur') {
    loadImage(data.id, '//i.imgur.com/' + data.imgur_id + 'l.jpg', function onload(img) {
      let fullSizeEl

      handleFreeze(function onFrozen(isFrozen) {
        if (isFrozen) {
          if (fullSizeEl) {
            fullSizeEl.parentNode.removeChild(fullSizeEl)
          }
        } else {
          if (!fullSizeEl) {
            fullSizeEl = document.createElement('img')
            fullSizeEl.src = '//i.imgur.com/' + data.imgur_id + '.jpg'
            fullSizeEl.style.width = img.style.width
            fullSizeEl.style.height = img.style.height
            fullSizeEl.className = 'cover'
          }
          document.body.appendChild(fullSizeEl)
        }
      })
    })
  } else if (data.kind === 'img') {
    const domain = data.url.match(/\/\/([^/]+)\//)
    if (!domain || !allowedImageDomains.hasOwnProperty(domain[1])) {
      return
    }

    loadImage(data.id, data.url, function onload(img) {
      // inspired by http://stackoverflow.com/a/4276742
      const canvasEl = document.createElement('canvas')
      const w = canvasEl.width = img.naturalWidth
      const h = canvasEl.height = img.naturalHeight
      canvasEl.getContext('2d').drawImage(img, 0, 0, w, h)
      canvasEl.style.width = img.style.width
      canvasEl.style.height = img.style.height
      canvasEl.className = 'cover'
      document.body.appendChild(canvasEl)

      handleFreeze(function onFreeze(isFrozen) {
        if (isFrozen) {
          document.body.appendChild(canvasEl)
        } else {
          canvasEl.parentNode.removeChild(canvasEl)
        }
      })
    })
  } else if (data.kind === 'youtube') {
    const embed = document.createElement('iframe')
    embed.src = '//www.youtube.com/embed/' + data.youtube_id + '?' + queryString.stringify({
      autoplay: data.autoplay,
      start: data.start,
    })
    document.body.appendChild(embed)
  }
}

render()
