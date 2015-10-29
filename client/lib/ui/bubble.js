import React from 'react/addons'
import classNames from 'classnames'
const ReactCSSTransitionGroup = React.addons.CSSTransitionGroup


export default React.createClass({
  displayName: 'Bubble',

  propTypes: {
    visible: React.PropTypes.bool,
    anchorEl: React.PropTypes.any,
    className: React.PropTypes.string,
    offset: React.PropTypes.func,
    edgeSpacing: React.PropTypes.number,
    onDismiss: React.PropTypes.func,
    children: React.PropTypes.node,
  },

  mixins: [require('react-immutable-render-mixin')],

  getDefaultProps() {
    return {
      edgeSpacing: 10,
    }
  },

  componentWillMount() {
    Heim.addEventListener(uidocument.body, Heim.isTouch ? 'touchstart' : 'click', this.onOutsideClick, false)
  },

  componentDidMount() {
    this.reposition()
  },

  componentDidUpdate() {
    this.reposition()
  },

  componentWillUnmount() {
    Heim.removeEventListener(uidocument.body, Heim.isTouch ? 'touchstart' : 'click', this.onOutsideClick, false)
  },

  onOutsideClick(ev) {
    if (this.props.visible && !this.getDOMNode().contains(ev.target) && this.props.onDismiss) {
      this.props.onDismiss(ev)
    }
  },

  reposition() {
    // FIXME: only handles left anchors. expand/complexify to work for multiple
    // orientations when necessary.
    if (this.props.visible && this.props.anchorEl) {
      const box = this.props.anchorEl.getBoundingClientRect()
      const node = this.refs.bubble.getDOMNode()

      let top = box.top
      top -= Math.max(0, top + node.clientHeight + this.props.edgeSpacing - uiwindow.innerHeight)

      let left = box.right

      if (this.props.offset) {
        const offsetBox = this.props.offset(box)
        left -= offsetBox.left || 0
        top -= offsetBox.top || 0
      }

      node.style.left = left + 'px'
      node.style.top = top + 'px'
    }
  },

  render() {
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
})
