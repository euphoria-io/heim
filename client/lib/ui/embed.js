var _ = require('lodash')
var React = require('react')
var classNames = require('classnames')
var EventEmitter = require('eventemitter3')
var queryString = require('querystring')

var actions = require('../actions')

var nextEmbedId = 0
var embedIndex = new EventEmitter()
actions.embedMessage.listen(data => embedIndex.emit(data.id, data))

module.exports = React.createClass({
  displayName: 'Embed',

  mixins: [
    require('react-immutable-render-mixin'),
  ],

  getInitialState: function() {
    return {
      width: null,
    }
  },

  componentWillMount: function() {
    this.embedId = nextEmbedId
    embedIndex.on(this.embedId, this.onMessage)
    nextEmbedId++
  },

  componentWillUnmount: function() {
    embedIndex.off(this.embedId, this.onMessage)
  },

  onMessage: function(msg) {
    if (msg.type == 'size') {
      this.setState({
        width: msg.data.width
      })
    }
  },

  _sendMessage: function(data) {
    this.refs.iframe.getDOMNode().contentWindow.postMessage(data, process.env.EMBED_ORIGIN)
  },

  freeze: function() {
    this._sendMessage({type: 'freeze'})
  },

  unfreeze: function() {
    this._sendMessage({type: 'unfreeze'})
  },

  render: function() {
    var data = _.extend({}, this.props, {id: this.embedId})
    delete data.className
    var url = process.env.EMBED_ORIGIN + '/?' + queryString.stringify(data)
    return <iframe key={url} ref="iframe" className={classNames('embed', this.props.className)} style={{width: this.state.width}} src={url} />
  },
})
