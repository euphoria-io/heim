var _ = require('lodash')
var React = require('react')
var Autolinker = require('autolinker')
var twemoji = require('twemoji')
var emojiIndex = require('emoji-annotation-to-unicode')

var chat = require('../stores/chat')
var hueHash = require('../hue-hash')

var emojiNames = _.filter(_.map(emojiIndex, (v, k) => v && _.escapeRegExp(k)))
var emojiNamesRe = new RegExp(':(' + emojiNames.join('|') + '):', 'g')

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
    // FIXME: replace with React splitting parser

    var html = _.escape(this.props.content)

    if (!this.props.onlyEmoji) {
      html = html.replace(/\B&amp;(\w+)(?=$|[^\w;])/g, function(match, name) {
        return React.renderToStaticMarkup(<a href={'/room/' + name} target="_blank">&amp;{name}</a>)
      })

      html = html.replace(chat.mentionRe, function(match, name) {
        var color = 'hsl(' + hueHash.hue(name) + ', 50%, 42%)'
        return React.renderToStaticMarkup(<span style={{color: color}} className="mention-nick">@{name}</span>)
      })
    }

    html = twemoji.replace(html, function(match, icon, variant) {
      if (variant == '\uFE0E') {
        return match
      }
      return React.renderToStaticMarkup(<div className={'emoji emoji-' + twemoji.convert.toCodePoint(icon)}>{icon}</div>)
    })

    html = html.replace(emojiNamesRe, function(match, name) {
      return React.renderToStaticMarkup(<div className={'emoji emoji-' + emojiIndex[name]}>{match}</div>)
    })

    if (!this.props.onlyEmoji) {
      html = autolinker.link(html)
    }

    return <span className={this.props.className} style={this.props.style} dangerouslySetInnerHTML={{
      __html: html
    }} />
  },
})
