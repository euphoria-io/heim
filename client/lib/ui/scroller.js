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
    this._onScroll = _.throttle(this.onScroll, 100)
    this._checkScroll = _.throttle(this.checkScroll, 150)
    this._finishScroll = _.debounce(this.finishScroll, 100)
    this._targetInView = false
    this._lastScrollHeight = 0
    this._lastScrollTop = 0
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
    var displayTop = node.offsetTop
    var displayHeight = node.offsetHeight

    var target = node.querySelector(this.props.target)
    var targetPos = target.getBoundingClientRect().top
    this._targetInView = targetPos >= displayTop + target.offsetHeight && targetPos < displayTop + displayHeight

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
      this._anchorPos = anchor && anchor.getBoundingClientRect().top
    }
    this._lastScrollTop = node.scrollTop
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
    var displayTop = node.offsetTop
    var displayHeight = node.offsetHeight
    var scrollHeight = node.scrollHeight
    var target = node.querySelector(this.props.target)
    var canScroll = displayHeight < scrollHeight

    var posRef, oldPos
    if (forceTargetInView || this._targetInView) {
      var hasGrown = scrollHeight > this._lastScrollHeight
      var fromBottom = scrollHeight - (node.scrollTop + displayHeight)
      var canScrollBottom = canScroll && fromBottom <= this.props.edgeSpace

      var targetBox = target.getBoundingClientRect()
      var targetPos = targetBox.top
      var clampedPos = clamp(displayTop + this.props.edgeSpace - targetBox.height, targetPos, displayTop + displayHeight - this.props.edgeSpace)

      var movingTowardsEdge = Math.sign(targetPos - this._anchorPos) != Math.sign(clampedPos - targetPos)
      var pastEdge = clampedPos != targetPos
      var movingPastEdge = movingTowardsEdge && pastEdge
      var jumping = Math.abs(targetPos - this._anchorPos) > 3 * target.offsetHeight

      var shouldHoldPos = hasGrown || (movingPastEdge && !jumping)
      var shouldScrollBottom = hasGrown && canScrollBottom

      posRef = target
      if (this._targetInView && shouldHoldPos && !shouldScrollBottom) {
        oldPos = this._anchorPos
      } else {
        if (forceTargetInView && !this._targetInView || shouldScrollBottom || jumping) {
          oldPos = clampedPos
        }
      }
    } else if (this._anchor) {
      // Otherwise, try to keep the anchor element in the same place it was when
      // we last saw it via updateAnchorPos.
      posRef = this._anchor
      oldPos = this._anchorPos
    }

    var delta = posRef.getBoundingClientRect().top - oldPos
    var scrollDelta = node.scrollTop - this._lastScrollTop
    if (delta && canScroll) {
      if (isWebkit) {
        // Note: mobile Webkit does this funny thing where getting/setting
        // scrollTop doesn't happen promptly during inertial scrolling. It turns
        // out that setting scrollTop inside a requestAnimationFrame callback
        // circumvents this issue.
        window.requestAnimationFrame(function() {
          // Time passes before the frame, so we need to update the deltas.
          delta = posRef.getBoundingClientRect().top - oldPos
          scrollDelta = node.scrollTop - this._lastScrollTop
          node.scrollTop += delta + scrollDelta
          this._lastScrollTop = node.scrollTop
        }.bind(this))
      } else {
        node.scrollTop += delta + scrollDelta
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
      <div ref="scroller" onScroll={this._onScroll} className={this.props.className}>
        {this.props.children}
      </div>
    )
  },
})
