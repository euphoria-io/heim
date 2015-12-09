import React from 'react'
import classNames from 'classnames'

import FastButton from './FastButton'
import ToggleBubble from './ToggleBubble'


export default React.createClass({
  displayName: 'RoomTitle',

  propTypes: {
    name: React.PropTypes.string.isRequired,
    connected: React.PropTypes.bool,
    authType: React.PropTypes.string,
  },

  mixins: [require('react-immutable-render-mixin')],

  showPrivacyInfo() {
    this.refs.privacyInfo.show()
  },

  render() {
    let className
    let caption
    let details

    if (this.props.connected === null) {
      caption = 'connecting...'
      details = 'waiting for server response'
    } else if (this.props.connected === false) {
      caption = 'reconnecting...'
      className = 'reconnecting'
      details = 'hang tight! we\'ll try again every few seconds until we get in.'
    } else {
      switch (this.props.authType) {
      case 'passcode':
        className = caption = 'private'
        details = 'this room requires a passcode for entry'
        break
      case 'public':
        className = caption = 'public'
        details = 'anyone with a link can join this room'
        break
      // no default
      }
    }

    return (
      <span className="room-container">
        <span className="room">
          <a className="name" href={'/room/' + this.props.name} onClick={ev => ev.preventDefault()}>&amp;{this.props.name}</a>
          <FastButton fastTouch className={classNames('state', className)} onClick={this.showPrivacyInfo}>{caption}</FastButton>
        </span>
        <ToggleBubble ref="privacyInfo" className="small-text privacy-info" visible={this.props.connected === false}>
          {details}
        </ToggleBubble>
      </span>
    )
  },
})
