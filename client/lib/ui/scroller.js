var _ = require('lodash')
var React = require('react')

var clamp = require('../clamp')


// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Math/sign
Math.sign = Math.sign || function(x) {
  x = +x  // convert to a number
  if (x === 0 || isNaN(x)) {
    return x
  }
  return x > 0 ? 1 : -1
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
    this._onScroll = _.throttle(this.onScroll, 100)
    this._checkScroll = _.throttle(this.checkScroll, 150)
    this._targetInView = false
    this._lastViewHeight = 0
    this._lastScrollHeight = 0
    this._lastScrollTop = 0
    this._anchor = null
    this._anchorPos = null
    this._waitingForUpdate = false
    this._lastTouch = 0
    this._animationFrames = {}
  },

  componentDidMount: function() {
    this.scroll({forceTargetInView: true})
    this.checkScrollbar()
  },

  componentWillUnmount: function() {
    this._onScroll.cancel()
    this._checkScroll.cancel()
  },

  _chromeRAFHack: function(id, callback) {
    if (Heim.isChrome && Heim.isTouch) {
      uiwindow.cancelAnimationFrame(this._animationFrames[id])
      this._animationFrames[id] = uiwindow.requestAnimationFrame(callback)
    } else {
      callback()
    }
  },

  onScroll: function() {
    this._checkScroll()
    this.updateAnchorPos()
    if (this.props.onScroll) {
      this.props.onScroll(this._isTouching())
    }
  },

  onFocusCapture: function() {
    // browser bugs and other difficult to account for shenanigans can cause
    // unwanted scrolls when inputs get focus. :( FIGHT BACK!
    // see https://code.google.com/p/chromium/issues/detail?id=437025
    setImmediate(() => {
      this.scroll({ignoreScrollDelta: true})
    })
  },

  onUpdate: function() {
    this.scroll()
    this.checkScrollbar()
  },

  update: function() {
    this._waitingForUpdate = false
    this.onUpdate()
  },

  updateAnchorPos: function() {
    // Record the position of our point of reference. Either the target (if
    // it's in view), or the centermost child element.
    var node = this.getDOMNode()
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
    var node = this.getDOMNode()

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

    var node = this.getDOMNode()

    if (this.props.onNearTop && node.scrollTop < node.scrollHeight / 8) {
      // since RAF doesn't execute while the page is hidden, scrolling in
      // infinite scroll won't occur in Chrome if users are on another tab.
      // this was causing an infinite loop: the log would continuously be
      // fetched since the scrollTop remained at 0.
      this._waitingForUpdate = true
      this._chromeRAFHack('checkScroll', this.props.onNearTop)
    }
  },

  scroll: function(options) {
    // Scroll so our point of interest (target or anchor) is in the right place.
    //
    // Desired behavior:
    //
    // If options.forceTargetInView is set, ensure that the target is onscreen.
    // If it is not, move it within edgeSpace of the top or bottom.
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

    // Note: mobile Webkit does this funny thing where getting/setting
    // scrollTop doesn't happen promptly during inertial scrolling. It turns
    // out that setting scrollTop inside a requestAnimationFrame callback
    // circumvents this issue.
    options = options || {}
    this._chromeRAFHack('scroll', () => {
      var node = this.getDOMNode()
      var nodeBox = dimensions(node)
      var viewTop = nodeBox.top
      var viewHeight = nodeBox.height
      var scrollHeight = node.scrollHeight
      var target = node.querySelector(this.props.target)
      var canScroll = viewHeight < scrollHeight
      var edgeSpace = Math.min(this.props.edgeSpace, viewHeight / 2)

      var posRef, oldPos
      if (target && (options.forceTargetInView || this._targetInView)) {
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
          if (options.forceTargetInView && !this._targetInView || shouldScrollBottom || jumping) {
            oldPos = clampedPos
          }
        }
      } else if (this._anchor) {
        // Otherwise, try to keep the anchor element in the same place it was when
        // we last saw it via updateAnchorPos.
        posRef = this._anchor
        oldPos = this._anchorPos
      }

      if (posRef) {
        var delta = dimensions(posRef, 'bottom') - oldPos
        if (delta && canScroll) {
          var scrollDelta = options.ignoreScrollDelta ? 0 : node.scrollTop - this._lastScrollTop
          this._lastScrollTop = node.scrollTop += delta + scrollDelta
        }
      }
      this.updateAnchorPos()
      this._checkScroll()
    })
  },

  scrollToTarget: function() {
    this.scroll({forceTargetInView: true})
  },

  onTouchStart: function() {
    this._lastTouch = true

    // prevent overscroll from bleeding out in Mobile Safari
    if (!Heim.isiOS) {
      return
    }

    // http://stackoverflow.com/a/14130056
    var node = this.getDOMNode()
    if (node.scrollTop === 0) {
      node.scrollTop = 1
    } else if (node.scrollHeight === node.scrollTop + node.offsetHeight) {
      node.scrollTop -= 1
    }
  },

  onTouchEnd: function() {
    this._lastTouch = new Date()
  },

  _isTouching: function() {
    return this._lastTouch === true || new Date() - this._lastTouch < 100
  },

  getPosition: function() {
    var node = this.getDOMNode()
    return node.scrollTop / (node.scrollHeight - node.clientHeight) || 1
  },

  render: function() {
    return (
      <div onScroll={this._onScroll} onFocusCapture={this.onFocusCapture} onTouchStart={this.onTouchStart} onTouchEnd={this.onTouchEnd} className={this.props.className}>
        {this.props.children}
      </div>
    )
  },
})
