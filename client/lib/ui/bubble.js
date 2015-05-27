var _ = require('lodash')
var React = require('react/addons')
var classNames = require('classnames')
var ReactCSSTransitionGroup = React.addons.CSSTransitionGroup


module.exports = React.createClass({
  displayName: 'Bubble',

  mixins: [require('react-immutable-render-mixin')],

  componentWillMount: function() {
    Heim.addEventListener(uidocument.body, Heim.isTouch ? 'touchstart' : 'click', this.onOutsideClick, false)
  },

  componentWillUnmount: function() {
    Heim.removeEventListener(uidocument.body, Heim.isTouch ? 'touchstart' : 'click', this.onOutsideClick, false)
  },

  onOutsideClick: function(ev) {
    if (!this.getDOMNode().contains(ev.target) && this.props.onDismiss) {
      this.props.onDismiss()
    }
  },

  render: function() {
    return (
      <ReactCSSTransitionGroup transitionName="bubble">
        {this.props.visible &&
          <div key="bubble" className={classNames('bubble', this.props.className)} style={{marginRight: this.props.rightOffset}}>
            {this.props.children}
          </div>
        }
      </ReactCSSTransitionGroup>
    )
  },
})
