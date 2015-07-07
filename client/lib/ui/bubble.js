var React = require('react/addons')
var classNames = require('classnames')
var ReactCSSTransitionGroup = React.addons.CSSTransitionGroup


module.exports = React.createClass({
  displayName: 'Bubble',

  mixins: [require('react-immutable-render-mixin')],

  getDefaultProps: function() {
    return {
      edgeSpacing: 10,
    }
  },

  componentWillMount: function() {
    Heim.addEventListener(uidocument.body, Heim.isTouch ? 'touchstart' : 'click', this.onOutsideClick, false)
  },

  componentWillUnmount: function() {
    Heim.removeEventListener(uidocument.body, Heim.isTouch ? 'touchstart' : 'click', this.onOutsideClick, false)
  },

  onOutsideClick: function(ev) {
    if (this.props.visible && !this.getDOMNode().contains(ev.target) && this.props.onDismiss) {
      this.props.onDismiss(ev)
    }
  },

  render: function() {
    return (
      <ReactCSSTransitionGroup transitionName="bubble">
        {this.props.visible &&
          <div ref="bubble" key="bubble" className={classNames('bubble', this.props.className)}>
            {this.props.children}
          </div>
        }
      </ReactCSSTransitionGroup>
    )
  },

  componentDidMount: function() {
    this.reposition()
  },

  componentDidUpdate: function() {
    this.reposition()
  },

  reposition: function() {
    // FIXME: only handles left anchors. expand/complexify to work for multiple
    // orientations when necessary.
    if (this.props.visible && this.props.anchorEl) {
      var box = this.props.anchorEl.getBoundingClientRect()
      var node = this.refs.bubble.getDOMNode()

      var top = box.top
      top -= Math.max(0, top + node.clientHeight + this.props.edgeSpacing - uiwindow.innerHeight)

      var left = box.right

      if (this.props.offset) {
        var offsetBox = this.props.offset()
        left -= offsetBox.left
        top -= offsetBox.top
      }

      node.style.left = left + 'px'
      node.style.top = top + 'px'
    }
  },
})
