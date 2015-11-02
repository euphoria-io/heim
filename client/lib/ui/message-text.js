import _ from 'lodash'
import React from 'react'
import ReactDOMServer from 'react-dom/server'
import Autolinker from 'autolinker'
import twemoji from 'twemoji'
import emoji from '../emoji'

import chat from '../stores/chat'
import hueHash from '../hue-hash'
import heimURL from '../heim-url'


const autolinker = new Autolinker({
  twitter: false,
  truncate: 40,
  replaceFn(self, match) {
    if (match.getType() === 'url') {
      const url = match.getUrl()
      const tag = self.getTagBuilder().build(match)

      if (/^javascript/.test(url.toLowerCase())) {
        // Thanks, Jordan!
        return false
      }

      if (location.protocol === 'https:' && RegExp('^https?:\/\/' + location.hostname).test(url)) {
        // self-link securely
        tag.setAttr('href', url.replace(/^http:/, 'https:'))
      } else {
        tag.setAttr('rel', 'noreferrer')
      }

      return tag
    }
  },
})

export default React.createClass({
  displayName: 'MessageText',

  propTypes: {
    content: React.PropTypes.string.isRequired,
    maxLength: React.PropTypes.number,
    onlyEmoji: React.PropTypes.bool,
    className: React.PropTypes.string,
    title: React.PropTypes.string,
    style: React.PropTypes.object,
  },

  mixins: [
    require('react-immutable-render-mixin'),
  ],

  render() {
    // FIXME: replace with React splitting parser + preserve links when trimmed

    let content = this.props.content

    if (this.props.maxLength) {
      content = _.trunc(content, this.props.maxLength)
    }

    let html = _.escape(content)

    if (!this.props.onlyEmoji) {
      html = html.replace(/\B&amp;(\w+)(?=$|[^\w;])/g, (match, name) =>
        ReactDOMServer.renderToStaticMarkup(<a href={heimURL('/room/' + name + '/')} target="_blank">&amp;{name}</a>)
      )

      html = html.replace(chat.mentionRe, (match, name) => {
        const color = 'hsl(' + hueHash.hue(name) + ', 50%, 42%)'
        return ReactDOMServer.renderToStaticMarkup(<span style={{color: color}} className="mention-nick">@{name}</span>)
      })
    }

    html = html.replace(emoji.namesRe, (match, name) =>
      ReactDOMServer.renderToStaticMarkup(<div className={'emoji emoji-' + emoji.index[name]} title={match}>{match}</div>)
    )

    html = twemoji.replace(html, (match, icon, variant) => {
      if (variant === '\uFE0E') {
        return match
      }
      const codePoint = emoji.lookupEmojiCharacter(icon)
      if (!codePoint) {
        return match
      }
      const emojiName = emoji.names[codePoint] && ':' + emoji.names[codePoint] + ':'
      return ReactDOMServer.renderToStaticMarkup(<div className={'emoji emoji-' + codePoint} title={emojiName}>{icon}</div>)
    })

    if (!this.props.onlyEmoji) {
      html = autolinker.link(html)
    }

    return (
      <span className={this.props.className} style={this.props.style} title={this.props.title} dangerouslySetInnerHTML={{
        __html: html,
      }} />
    )
  },
})
