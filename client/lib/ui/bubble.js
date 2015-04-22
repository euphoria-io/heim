var _ = require('lodash')
var React = require('react/addons')
var classNames = require('classnames')
var ReactCSSTransitionGroup = React.addons.CSSTransitionGroup


module.exports = React.createClass({
  displayName: 'Bubble',

  mixins: [require('react-immutable-render-mixin')],

  getInitialState: function() {
    return {visible: false}
  },

  componentWillMount: function() {
    Heim.addEventListener(uidocument.body, Heim.isTouch ? 'touchstart' : 'click', this.onOutsideClick, false)
    this._hide = _.debounce(this.hide, 0)
  },

  componentWillUnmount: function() {
    Heim.removeEventListener(uidocument.body, Heim.isTouch ? 'touchstart' : 'click', this.onOutsideClick, false)
  },

  show: function() {
    this.setState({visible: true})
    this._hide.cancel()
  },

  hide: function() {
    this.setState({visible: false})
  },

  toggle: function() {
    if (this.state.visible) {
      this._hide()
    } else {
      this.show()
    }
  },

  onOutsideClick: function(ev) {
    // queue hide so that if the click triggers a show, we don't hide and then
    // immediately reshow.
    if (!this.getDOMNode().contains(ev.target)) {
      this._hide()
    }
  },

  render: function() {
    return (
      <ReactCSSTransitionGroup transitionName="bubble">
        {this.state.visible &&
          <div key="bubble" className={classNames('bubble', this.props.className)} style={{marginRight: this.props.rightOffset}}>
            {this.props.children}
          </div>
        }
      </ReactCSSTransitionGroup>
    )
  },
})
