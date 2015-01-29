var _ = require('lodash')
var React = require('react')


// http://stackoverflow.com/a/16459606
var isWebkit = 'WebkitAppearance' in document.documentElement.style

function clamp(min, v, max) {
  return Math.min(Math.max(min, v), max)
}

module.exports = React.createClass({
  displayName: 'Scroller',

  componentWillMount: function() {
    window.addEventListener('resize', this.onResize)
    this._checkScroll = _.throttle(this.checkScroll, 150)
    this._finishScroll = _.debounce(this.finishScroll, 100)
    this._targetInView = false
    this._lastScrollHeight = 0
    this._anchor = null
    this._anchorPos = null
    this._scrollQueued = false
    this._waitingForUpdate = false
  },

  componentDidMount: function() {
    this.updateAnchorPos()
  },

  componentWillUnmount: function() {
    window.removeEventListener('resize', this.onResize)
  },

  onResize: function() {
    // When resizing, the goal is to keep the entry onscreen in the same
    // position, if possible. This is accomplished by scrolling relative to the
    // previous display height factored into the pos recorded by updateAnchorPos.
    this.scroll()
  },

  finishScroll: function() {
    this._scrollQueued = false
    this.updateAnchorPos()
  },

  onScroll: function() {
    this._checkScroll()
    this.updateAnchorPos()
  },

  componentDidUpdate: function() {
    this.scroll()
    this.updateAnchorPos()
    this.checkScrollbar()
    this._checkScroll()
    this._waitingForUpdate = false
  },

  updateAnchorPos: function() {
    if (this._scrollQueued) {
      // If we're waiting on a scroll, re-measuring the anchor position may
      // lose track of it if we're in the process of scrolling it onscreen.
      return
    }

    // Record the position of our point of reference. Either the target (if
    // it's in view), or the centermost child element.
    var node = this.refs.scroller.getDOMNode()
    var displayHeight = node.offsetHeight

    var target = node.querySelector(this.props.target)
    var targetPos = node.scrollTop + displayHeight - target.offsetTop
    this._targetInView = targetPos >= target.offsetHeight && targetPos < displayHeight

    var anchor
    if (this._targetInView) {
      this._anchor = target
      this._anchorPos = targetPos
    } else {
      var box = this.getDOMNode().getBoundingClientRect()
      anchor = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2)
      if (!anchor) {
        console.warn('scroller: unable to find anchor')  // jshint ignore:line
      }
      this._anchor = anchor
      this._anchorPos = anchor && node.scrollTop + displayHeight - anchor.offsetTop
    }
    this._lastScrollHeight = node.scrollHeight
  },

  checkScrollbar: function() {
    var node = this.refs.scroller.getDOMNode()

    if (this.props.onScrollbarSize) {
      var scrollbarWidth = node.offsetWidth - node.clientWidth
      if (scrollbarWidth != this.scrollbarWidth) {
        this.scrollbarWidth = scrollbarWidth
        this.props.onScrollbarSize(scrollbarWidth)
      }
    }
  },

  checkScroll: function() {
    if (this._waitingForUpdate) {
      return
    }

    var node = this.refs.scroller.getDOMNode()

    var displayHeight = node.offsetHeight
    if (this.props.onNearTop && node.scrollTop < displayHeight * 2) {
      this.props.onNearTop()
      this._waitingForUpdate = true
    }
  },

  scroll: function(forceTargetInView) {
    // Scroll so our point of interest (target or anchor) is in the right place.
    //
    // Desired behavior:
    //
    // If forceTargetInView is set, ensure that the target is onscreen. If it
    // is not, move it within edgeSpace of the top or bottom.
    //
    // If the target was previously in view, we want to ensure it still is. If
    // we're at the bottom of the page, new content should be able to push the
    // target up to edgeSpace. If we're jumping several rows, we want to make
    // sure we end up within edgeSpace. Otherwise, movements that would take us
    // past edgeSpace should scroll to keep the target within edgeSpace.
    //
    // If the target was not previously in view, maintain the position of the
    // anchor element.
    //
    var node = this.refs.scroller.getDOMNode()
    var displayHeight = node.offsetHeight
    var scrollHeight = node.scrollHeight
    var target = node.querySelector(this.props.target)
    var canScroll = displayHeight < scrollHeight

    var newScrollTop = null
    if (forceTargetInView || this._targetInView) {
      var hasGrown = scrollHeight > this._lastScrollHeight
      var fromBottom = scrollHeight - (node.scrollTop + displayHeight)
      var canScrollBottom = canScroll && fromBottom <= this.props.edgeSpace

      var targetPos = node.scrollTop + displayHeight - target.offsetTop
      var clampedPos = clamp(this.props.edgeSpace, targetPos, displayHeight - this.props.edgeSpace + target.offsetHeight)

      var movingTowardsEdge = Math.sign(targetPos - this._anchorPos) != Math.sign(clampedPos - targetPos)
      var pastEdge = clampedPos != targetPos
      var movingPastEdge = movingTowardsEdge && pastEdge
      var jumping = Math.abs(targetPos - this._anchorPos) > 3 * target.offsetHeight

      var shouldHoldPos = hasGrown || (movingPastEdge && !jumping)
      var shouldScrollBottom = hasGrown && canScrollBottom

      if (this._targetInView && shouldHoldPos && !shouldScrollBottom) {
        targetPos = this._anchorPos
      } else {
        if (forceTargetInView && !this._targetInView || shouldScrollBottom || jumping) {
          targetPos = clampedPos
        }
      }
      newScrollTop = targetPos - displayHeight + target.offsetTop
    } else if (this._anchor) {
      // Otherwise, try to keep the anchor element in the same place it was when
      // we last saw it via updateAnchorPos.
      newScrollTop = this._anchorPos - displayHeight + this._anchor.offsetTop
    }

    if (newScrollTop != node.scrollTop && canScroll) {
      if (isWebkit) {
        // Note: mobile Webkit does this funny thing where getting/setting
        // scrollTop doesn't happen promptly during inertial scrolling. It turns
        // out that setting scrollTop inside a requestAnimationFrame callback
        // circumvents this issue.
        window.requestAnimationFrame(function() {
          node.scrollTop = newScrollTop
        })
      } else {
        node.scrollTop = newScrollTop
      }
      this._scrollQueued = true
      this._finishScroll()
    }
  },

  scrollToTarget: function() {
    this.scroll(true)
  },

  render: function() {
    return (
      <div ref="scroller" onScroll={this.onScroll} className={this.props.className}>
        {this.props.children}
      </div>
    )
  },
})
