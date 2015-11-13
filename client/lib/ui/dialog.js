import React from 'react'
import classNames from 'classnames'

import Popup from './popup'
import FastButton from './fast-button'
import Spinner from './spinner'


export default React.createClass({
  displayName: 'Dialog',

  propTypes: {
    title: React.PropTypes.string,
    working: React.PropTypes.bool,
    onClose: React.PropTypes.func,
    className: React.PropTypes.string,
    children: React.PropTypes.node,
  },

  onShadeClick(ev) {
    if (ev.target === this.refs.shade) {
      this.props.onClose()
    }
  },

  render() {
    return (
      <div className="dim-shade dialog-cover fill" ref="shade" onClick={this.onShadeClick}>
        <Popup className={classNames('dialog', this.props.className)}>
          <div className="top-line">
            <div className="logo">
              <div className="emoji emoji-euphoria" />
              euphoria
            </div>
            <div className="title">{this.props.title}</div>
            <Spinner visible={this.props.working} />
            <div className="spacer" />
            <FastButton className="close" onClick={this.props.onClose} />
          </div>
          {this.props.children}
        </Popup>
      </div>
    )
  },
})
