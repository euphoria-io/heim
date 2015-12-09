import React from 'react'
import classNames from 'classnames'


export default React.createClass({
  displayName: 'ErrorMessage',

  propTypes: {
    name: React.PropTypes.string.isRequired,
    message: React.PropTypes.string,
    error: React.PropTypes.bool,
    className: React.PropTypes.string,
  },

  render() {
    return <div className={classNames('message', this.props.error && 'error', this.props.className)}>{this.props.message}</div>
  },
})
