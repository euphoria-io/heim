import _ from 'lodash'
import React from 'react'
import classNames from 'classnames'
import EventEmitter from 'eventemitter3'
import queryString from 'querystring'

import actions from '../actions'

let nextEmbedId = 0
const embedIndex = new EventEmitter()
actions.embedMessage.listen(data => embedIndex.emit(data.id, data))

export default React.createClass({
  displayName: 'Embed',

  propTypes: {
    className: React.PropTypes.string,
  },

  mixins: [
    require('react-immutable-render-mixin'),
  ],

  getInitialState() {
    return {
      width: null,
    }
  },

  componentWillMount() {
    this.embedId = nextEmbedId
    embedIndex.on(this.embedId, this.onMessage)
    nextEmbedId++
  },

  componentWillUnmount() {
    embedIndex.off(this.embedId, this.onMessage)
  },

  onMessage(msg) {
    if (msg.type === 'size') {
      this.setState({
        width: msg.data.width,
      })
    }
  },

  _sendMessage(data) {
    this.refs.iframe.getDOMNode().contentWindow.postMessage(data, process.env.EMBED_ORIGIN)
  },

  freeze() {
    this._sendMessage({type: 'freeze'})
  },

  unfreeze() {
    this._sendMessage({type: 'unfreeze'})
  },

  render() {
    const data = _.extend({}, this.props, {id: this.embedId})
    delete data.className
    const url = process.env.EMBED_ORIGIN + '/?' + queryString.stringify(data)
    return <iframe key={url} ref="iframe" className={classNames('embed', this.props.className)} style={{width: this.state.width}} src={url} />
  },
})
