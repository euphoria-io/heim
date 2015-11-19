import _ from 'lodash'
import React from 'react'
import ReactDOMServer from 'react-dom/server'

import findParent from '../find-parent'
import domWalkForward from '../dom-walk-forward'
import emoji from '../emoji'


export default function handleCopy(ev) {
  const selection = uiwindow.getSelection()
  const range = selection.getRangeAt(0)

  // first, if the selection start and end are within the same message
  // line, do nothing.
  function findParentMessageLine(el) {
    return findParent(el, el2 => el2.classList && (el2.classList.contains('line') || el2.classList.contains('message-node')))
  }
  const startMessageEl = findParentMessageLine(range.startContainer)
  const endMessageEl = findParentMessageLine(range.endContainer)
  if (!startMessageEl || !endMessageEl) {
    return
  }

  const contentEl = startMessageEl.querySelector('.content')
  if (startMessageEl && contentEl.contains(range.startContainer) && contentEl.contains(range.endContainer)) {
    return
  }

  const entryEl = startMessageEl.querySelector('.entry')
  if (entryEl && entryEl.contains(range.startContainer)) {
    return
  }

  const messageEls = []
  let minDepth
  domWalkForward(startMessageEl, endMessageEl, el => {
    if (!el.classList || !el.classList.contains('line')) {
      return
    }
    messageEls.push(el)
    const depth = el.parentNode.dataset.depth
    if (!minDepth || depth < minDepth) {
      minDepth = depth
    }
  })

  function formatEmoji(content) {
    return content.replace(emoji.namesRe, (match, name) => emoji.nameToUnicode(name) || match)
  }

  const textParts = []
  const htmlLines = []
  _.each(messageEls, lineEl => {
    const el = lineEl.parentNode
    const messageId = el.dataset.messageId
    const message = Heim.chat.store.state.messages.get(messageId)
    if (!message) {
      return
    }

    const preContentItems = []
    preContentItems.push(_.repeat(' ', 2 * (el.dataset.depth - minDepth)))
    preContentItems.push('[')
    preContentItems.push(message.getIn(['sender', 'name']))
    preContentItems.push('] ')
    const preContent = preContentItems.join('')
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
  ev.clipboardData.setData('text/html', ReactDOMServer.renderToStaticMarkup(<div className="heim-messages">{htmlLines}</div>))
  ev.preventDefault()
}
