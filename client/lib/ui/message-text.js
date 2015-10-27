var _ = require('lodash')
var React = require('react')
var Autolinker = require('autolinker')
var twemoji = require('twemoji')
var emoji = require('../emoji')

var chat = require('../stores/chat')
var hueHash = require('../hue-hash')
var heimURL = require('../heim-url')


var autolinker = new Autolinker({
  twitter: false,
  truncate: 40,
  replaceFn: function(autolinker, match) {
    if (match.getType() == 'url') {
      var url = match.getUrl()
      var tag = autolinker.getTagBuilder().build(match)

      if (/^javascript/.test(url.toLowerCase())) {
        // Thanks, Jordan!
        return false
      }

      if (location.protocol == 'https:' && RegExp('^https?:\/\/' + location.hostname).test(url)) {
        // self-link securely
        tag.setAttr('href', url.replace(/^http:/, 'https:'))
      } else {
        tag.setAttr('rel', 'noreferrer')
      }

      return tag
    }
  },
})

module.exports = React.createClass({
  displayName: 'MessageText',

  mixins: [
    require('react-immutable-render-mixin'),
  ],

  render: function() {
    // FIXME: replace with React splitting parser + preserve links when trimmed

    var content = this.props.content

    if (this.props.maxLength) {
      content = _.trunc(content, this.props.maxLength)
    }

    var html = _.escape(content)

    if (!this.props.onlyEmoji) {
      html = html.replace(/\B&amp;(\w+)(?=$|[^\w;])/g, function(match, name) {
        return React.renderToStaticMarkup(<a href={heimURL('/room/' + name + '/')} target="_blank">&amp;{name}</a>)
      })

      html = html.replace(chat.mentionRe, function(match, name) {
        var color = 'hsl(' + hueHash.hue(name) + ', 50%, 42%)'
        return React.renderToStaticMarkup(<span style={{color: color}} className="mention-nick">@{name}</span>)
      })
    }

    html = html.replace(emoji.namesRe, function(match, name) {
      return React.renderToStaticMarkup(<div className={'emoji emoji-' + emoji.index[name]} title={match}>{match}</div>)
    })

    html = twemoji.replace(html, function(match, icon, variant) {
      if (variant == '\uFE0E') {
        return match
      }
      var codePoint = emoji.lookupEmojiCharacter(icon)
      if (!codePoint) {
        return match
      }
      var emojiName = emoji.names[codePoint] && ':' + emoji.names[codePoint] + ':'
      return React.renderToStaticMarkup(<div className={'emoji emoji-' + codePoint} title={emojiName}>{icon}</div>)
    })

    if (!this.props.onlyEmoji) {
      html = autolinker.link(html)
    }

    return <span className={this.props.className} style={this.props.style} title={this.props.title} dangerouslySetInnerHTML={{
      __html: html
    }} />
  },
})
