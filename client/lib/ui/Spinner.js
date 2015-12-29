import React from 'react'
import ReactCSSTransitionGroup from 'react-addons-css-transition-group'


module.exports = React.createClass({
  displayName: 'Spinner',

  propTypes: {
    visible: React.PropTypes.bool,
  },

  getDefaultProps() {
    return {visible: true}
  },

  render() {
    return <ReactCSSTransitionGroup transitionName="spinner" transitionEnterTimeout={100} transitionLeaveTimeout={100}>{this.props.visible && <div key="spinner" className="spinner" />}</ReactCSSTransitionGroup>
  },
})

