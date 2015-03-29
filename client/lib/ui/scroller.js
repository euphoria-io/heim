var _ = require('lodash')
var React = require('react')


// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Math/sign
Math.sign = Math.sign || function(x) {
  x = +x  // convert to a number
  if (x === 0 || isNaN(x)) {
    return x
  }
  return x > 0 ? 1 : -1
}

function clamp(min, v, max) {
  return Math.min(Math.max(min, v), max)
}

function dimensions(el, prop) {
  var rect = el.getBoundingClientRect()
  if (prop) {
    return Math.round(rect[prop])
  } else {
    var dims = {}
    // would like to use _.mapValues here, but doesn't work in Firefox (not
    // "own" properties?)
    _.forIn(rect, function(v, k) {
      dims[k] = Math.round(v)
    })
    return dims
  }
}

module.exports = React.createClass({
  displayName: 'Scroller',

  componentWillMount: function() {
    Heim.addEventListener(uiwindow, 'resize', this.onResize)
    this._onScroll = _.throttle(this.onScroll, 100)
    this._checkScroll = _.throttle(this.checkScroll, 150)
    this._finishScroll = _.debounce(this.finishScroll, 100)
    this._targetInView = false
    this._lastViewHeight = 0
    this._lastScrollHeight = 0
    this._lastScrollTop = 0
    this._anchor = null
    this._anchorPos = null
    this._scrollQueued = false
    this._waitingForUpdate = false
    this._lastTouch = 0
  },

  componentDidMount: function() {
    this.updateAnchorPos()
    this.onResize()
  },

  componentWillUnmount: function() {
    Heim.removeEventListener(uiwindow, 'resize', this.onResize)
  },

  onResize: function() {
    // When resizing, the goal is to keep the entry onscreen in the same
    // position, if possible. This is accomplished by scrolling relative to the
    // previous display height factored into the pos recorded by updateAnchorPos.
    this.scroll()
    if (this.props.onResize) {
      var node = this.refs.scroller.getDOMNode()
      this.props.onResize(node.offsetWidth, node.offsetHeight)
    }
  },

  finishScroll: function() {
    this._scrollQueued = false
    this.updateAnchorPos()
  },

  onScroll: function() {
    this._checkScroll()
    this.updateAnchorPos()
    if (!this._scrollQueued) {
      this.props.onScroll(new Date() - this._lastTouch < 100)
    }
  },

  onUpdate: function() {
    this.scroll()
    this.updateAnchorPos()
    this.checkScrollbar()
    this._checkScroll()
  },

  componentDidUpdate: function() {
    this.onUpdate()
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
    var nodeBox = dimensions(node)
    var viewTop = nodeBox.top
    var viewHeight = nodeBox.height

    var target = node.querySelector(this.props.target)
    var targetPos
    if (target) {
      targetPos = dimensions(target, 'bottom')
      this._targetInView = targetPos >= viewTop - 5 + target.offsetHeight && targetPos <= viewTop + viewHeight + 5
    } else {
      this._targetInView = false
    }

    var anchor
    if (this._targetInView) {
      this._anchor = target
      this._anchorPos = targetPos
    } else {
      var box = dimensions(this.getDOMNode())
      anchor = uidocument.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2)
      if (!anchor) {
        console.warn('scroller: unable to find anchor')  // jshint ignore:line
      }
      this._anchor = anchor
      this._anchorPos = anchor && dimensions(anchor, 'bottom')
    }
    this._lastScrollTop = node.scrollTop
    this._lastScrollHeight = node.scrollHeight
    this._lastViewHeight = viewHeight
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

    if (this.props.onNearTop && node.scrollTop < node.scrollHeight / 8) {
      if (Heim.isChrome) {
        // since RAF doesn't execute while the page is hidden, scrolling in
        // infinite scroll won't occur in Chrome if users are on another tab.
        // this was causing an infinite loop: the log would continuously be
        // fetched since the scrollTop remained at 0.
        uiwindow.requestAnimationFrame(this.props.onNearTop)
      } else {
        this.props.onNearTop()
      }
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
    var nodeBox = dimensions(node)
    var viewTop = nodeBox.top
    var viewHeight = nodeBox.height
    var scrollHeight = node.scrollHeight
    var target = node.querySelector(this.props.target)
    var canScroll = viewHeight < scrollHeight
    var edgeSpace = Math.min(this.props.edgeSpace, viewHeight / 2)

    var posRef, oldPos
    if (target && (forceTargetInView || this._targetInView)) {
      var viewShrunk = viewHeight < this._lastViewHeight
      var hasGrown = scrollHeight > this._lastScrollHeight
      var fromBottom = scrollHeight - (node.scrollTop + viewHeight)
      var canScrollBottom = canScroll && fromBottom <= edgeSpace

      var targetBox = dimensions(target)
      var targetPos = targetBox.bottom
      var clampedPos = clamp(viewTop + edgeSpace - targetBox.height, targetPos, viewTop + viewHeight - edgeSpace)

      var movingTowardsEdge = Math.sign(targetPos - this._anchorPos) != Math.sign(clampedPos - targetPos)
      var pastEdge = clampedPos != targetPos
      var movingPastEdge = movingTowardsEdge && pastEdge
      var jumping = Math.abs(targetPos - this._anchorPos) > 3 * target.offsetHeight

      var shouldHoldPos = hasGrown || (movingPastEdge && !jumping)
      var shouldScrollBottom = hasGrown && canScrollBottom || viewShrunk

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

    var delta = dimensions(posRef, 'bottom') - oldPos
    var scrollDelta = node.scrollTop - this._lastScrollTop
    if (delta && canScroll) {
      if (Heim.isChrome) {
        // Note: mobile Webkit does this funny thing where getting/setting
        // scrollTop doesn't happen promptly during inertial scrolling. It turns
        // out that setting scrollTop inside a requestAnimationFrame callback
        // circumvents this issue.
        uiwindow.requestAnimationFrame(function() {
          // Time passes before the frame, so we need to update the deltas.
          delta = dimensions(posRef, 'bottom') - oldPos
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

  onTouchStart: function() {
    // prevent overscroll from bleeding out in Mobile Safari
    if (!Heim.isiOS) {
      return
    }

    // http://stackoverflow.com/a/14130056
    var node = this.refs.scroller.getDOMNode()
    if (node.scrollTop === 0) {
      node.scrollTop = 1
    } else if (node.scrollHeight === node.scrollTop + node.offsetHeight) {
      node.scrollTop -= 1
    }
  },

  onTouchEnd: function() {
    this._lastTouch = new Date()
  },

  render: function() {
    return (
      <div ref="scroller" onScroll={this._onScroll} onLoad={this.onUpdate} onTouchStart={this.onTouchStart} onTouchEnd={this.onTouchEnd} className={this.props.className}>
        {this.props.children}
      </div>
    )
  },
})
