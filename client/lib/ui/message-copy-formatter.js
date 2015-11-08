var _ = require('lodash')
var React = require('react/addons')

var findParent = require('../find-parent')
var domWalkForward = require('../dom-walk-forward')
var emoji = require('../emoji')


module.exports = function handleCopy(ev) {
  var selection = uiwindow.getSelection()
  if (selection.rangeCount === 0) {
    return
  }
  var range = selection.getRangeAt(0)

  // first, if the selection start and end are within the same message
  // line, do nothing.
  function findParentMessageLine(el) {
    return findParent(el, el => el.classList && (el.classList.contains('line') || el.classList.contains('message-node')))
  }
  var startMessageEl = findParentMessageLine(range.startContainer)
  var endMessageEl = findParentMessageLine(range.endContainer)
  if (!startMessageEl || !endMessageEl) {
    return
  }

  var contentEl = startMessageEl.querySelector('.content')
  if (startMessageEl && contentEl.contains(range.startContainer) && contentEl.contains(range.endContainer)) {
    return
  }

  var entryEl = startMessageEl.querySelector('.entry')
  if (entryEl && entryEl.contains(range.startContainer)) {
    return
  }

  var messageEls = []
  var minDepth
  domWalkForward(startMessageEl, endMessageEl, function(el) {
    if (!el.classList || !el.classList.contains('line')) {
      return
    }
    messageEls.push(el)
    var depth = el.parentNode.dataset.depth
    if (!minDepth || depth < minDepth) {
      minDepth = depth
    }
  })

  function formatEmoji(content) {
    return content.replace(emoji.namesRe, (match, name) => emoji.nameToUnicode(name) || match)
  }

  var textParts = []
  var htmlLines = []
  _.each(messageEls, lineEl => {
    var el = lineEl.parentNode
    var messageId = el.dataset.messageId
    var message = Heim.chat.store.state.messages.get(messageId)

    var preContent = []
    preContent.push(_.repeat(' ', 2 * (el.dataset.depth - minDepth)))
    preContent.push('[')
    preContent.push(message.getIn(['sender', 'name']))
    preContent.push('] ')
    preContent = preContent.join('')
    textParts.push(preContent)
    textParts.push(message.get('content').trim().replace(/\n/g, '\n' + _.repeat(' ', preContent.length)))
    textParts.push('\n')

    htmlLines.push(
      <div key={message.get('id')} data-message-id={message.get('id')} style={{
        padding: '2px 0',
        marginLeft: 16 * (el.dataset.depth - minDepth),
        lineHeight: '1.25em',
      }}>
        <span className="nick" style={{
          display: 'inline-block',
          padding: '0 4px',
          marginRight: '6px',
          background: 'hsl(' + message.getIn(['sender', 'hue']) + ', 65%, 85%)',
          borderRadius: '2px',
        }}>{formatEmoji(message.getIn(['sender', 'name']))} </span>
        {formatEmoji(message.get('content').trim())}
      </div>
    )
  })

  ev.clipboardData.setData('text/plain', formatEmoji(textParts.join('')).trim())
  ev.clipboardData.setData('text/html', React.renderToStaticMarkup(<div className="heim-messages">{htmlLines}</div>))
  ev.preventDefault()
}
