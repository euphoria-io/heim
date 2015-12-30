import _ from 'lodash'
import React from 'react'
import ReactDOM from 'react-dom'

import clamp from '../clamp'


function dimensions(el, prop) {
  const rect = el.getBoundingClientRect()
  if (prop) {
    return Math.round(rect[prop])
  }
  const dims = {}
  // would like to use _.mapValues here, but doesn't work in Firefox (not
  // "own" properties?)
  _.forIn(rect, (v, k) => {
    dims[k] = Math.round(v)
  })
  return dims
}

export default React.createClass({
  displayName: 'Scroller',

  propTypes: {
    onScroll: React.PropTypes.func,
    onNearTop: React.PropTypes.func,
    onScrollbarSize: React.PropTypes.func,
    target: React.PropTypes.string,
    edgeSpace: React.PropTypes.number,
    className: React.PropTypes.string,
    children: React.PropTypes.node,
  },

  componentWillMount() {
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

  componentDidMount() {
    this.scroll({forceTargetInView: true})
    this.checkScrollbar()
  },

  componentWillUnmount() {
    this._onScroll.cancel()
    this._checkScroll.cancel()
  },

  onScroll() {
    this._chromeRAFHack('onScroll', () => {
      this._checkScroll()
      this.updateAnchorPos()
      if (this.props.onScroll) {
        this.props.onScroll(this._isTouching())
      }
    })
  },

  onFocusCapture() {
    // browser bugs and other difficult to account for shenanigans can cause
    // unwanted scrolls when inputs get focus. :( FIGHT BACK!
    // see https://code.google.com/p/chromium/issues/detail?id=437025
    setImmediate(() => {
      this.scroll({ignoreScrollDelta: true})
    })
  },

  onUpdate() {
    this.scroll()
    this.checkScrollbar()
  },

  onTouchStart() {
    this._lastTouch = true

    // prevent overscroll from bleeding out in Mobile Safari
    if (!Heim.isiOS) {
      return
    }

    // http://stackoverflow.com/a/14130056
    const node = ReactDOM.findDOMNode(this)
    if (node.scrollTop === 0) {
      node.scrollTop = 1
    } else if (node.scrollHeight === node.scrollTop + node.offsetHeight) {
      node.scrollTop -= 1
    }
  },

  onTouchEnd() {
    this._lastTouch = new Date()
  },

  getPosition() {
    const node = ReactDOM.findDOMNode(this)
    if (!node.scrollHeight) {
      return false
    }

    const frac = node.scrollTop / (node.scrollHeight - node.clientHeight)
    return frac ? Math.round(frac * 100) / 100 : 1
  },

  _chromeRAFHack(id, callback, immediate) {
    if (!immediate && Heim.isChrome && Heim.isTouch) {
      if (this._animationFrames[id]) {
        return
      }

      this._animationFrames[id] = uiwindow.requestAnimationFrame(() => {
        this._animationFrames[id] = null
        callback()
      })
    } else {
      callback()
    }
  },

  update() {
    this._waitingForUpdate = false
    this.onUpdate()
  },

  updateAnchorPos() {
    // Record the position of our point of reference. Either the target (if
    // it's in view), or the centermost child element.
    const node = ReactDOM.findDOMNode(this)
    const nodeBox = dimensions(node)
    const viewTop = nodeBox.top
    const viewHeight = nodeBox.height

    const target = node.querySelector(this.props.target)
    let targetPos
    if (target) {
      targetPos = dimensions(target, 'bottom')
      this._targetInView = targetPos >= viewTop - 5 + target.offsetHeight && targetPos <= viewTop + viewHeight + 5
    } else {
      this._targetInView = false
    }

    let anchor
    if (this._targetInView) {
      this._anchor = target
      this._anchorPos = targetPos
    } else {
      const box = dimensions(node)
      const bodyBox = dimensions(uidocument.body)
      const boxRight = Math.min(box.right, bodyBox.right)
      anchor = uidocument.elementFromPoint(boxRight - 40, box.top + box.height / 2)
      if (!anchor) {
        // FIXME: this can happen from time to time, need a better fallback
      }
      this._anchor = anchor
      this._anchorPos = anchor && dimensions(anchor, 'bottom')
    }
    this._lastScrollTop = node.scrollTop
    this._lastScrollHeight = node.scrollHeight
    this._lastViewHeight = viewHeight
  },

  checkScrollbar() {
    const node = ReactDOM.findDOMNode(this)

    if (this.props.onScrollbarSize) {
      const scrollbarWidth = node.offsetWidth - node.clientWidth
      if (scrollbarWidth !== this.scrollbarWidth) {
        this.scrollbarWidth = scrollbarWidth
        this.props.onScrollbarSize(scrollbarWidth)
      }
    }
  },

  checkScroll() {
    if (this._waitingForUpdate) {
      return
    }

    const node = ReactDOM.findDOMNode(this)

    if (node.scrollHeight === 0) {
      return
    }

    if (this.props.onNearTop && node.scrollTop < node.scrollHeight / 8) {
      // since RAF doesn't execute while the page is hidden, scrolling in
      // infinite scroll won't occur in Chrome if users are on another tab.
      // this was causing an infinite loop: the log would continuously be
      // fetched since the scrollTop remained at 0.
      this._waitingForUpdate = true
      this._chromeRAFHack('checkScroll', this.props.onNearTop)
    }
  },

  scroll(options = {}) {
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
    this._chromeRAFHack('scroll', () => {
      const node = ReactDOM.findDOMNode(this)
      const nodeBox = dimensions(node)
      const viewTop = nodeBox.top
      const viewHeight = nodeBox.height
      const scrollHeight = node.scrollHeight
      const target = node.querySelector(this.props.target)
      const canScroll = viewHeight < scrollHeight
      const edgeSpace = Math.min(this.props.edgeSpace, viewHeight / 2)

      let posRef
      let oldPos
      if (target && (options.forceTargetInView || this._targetInView)) {
        const viewShrunk = viewHeight < this._lastViewHeight
        const hasGrown = scrollHeight > this._lastScrollHeight
        const fromBottom = scrollHeight - (node.scrollTop + viewHeight)
        const canScrollBottom = canScroll && fromBottom <= edgeSpace

        const targetBox = dimensions(target)
        const targetPos = targetBox.bottom
        const clampedPos = clamp(viewTop + edgeSpace - targetBox.height, targetPos, viewTop + viewHeight - edgeSpace)

        const movingTowardsEdge = Math.sign(targetPos - this._anchorPos) !== Math.sign(clampedPos - targetPos)
        const pastEdge = clampedPos !== targetPos
        const movingPastEdge = movingTowardsEdge && pastEdge
        const jumping = Math.abs(targetPos - this._anchorPos) > 3 * target.offsetHeight

        const shouldHoldPos = hasGrown || (movingPastEdge && !jumping)
        const shouldScrollBottom = hasGrown && canScrollBottom || viewShrunk

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
        const delta = dimensions(posRef, 'bottom') - oldPos
        if (delta && canScroll) {
          const scrollDelta = options.ignoreScrollDelta ? 0 : node.scrollTop - this._lastScrollTop
          this._lastScrollTop = node.scrollTop += delta + scrollDelta
        }
      }
      this.updateAnchorPos()
      this._checkScroll()
    }, options.immediate)
  },

  scrollToTarget(options = {}) {
    options.forceTargetInView = true
    this.scroll(options)
  },

  _isTouching() {
    return this._lastTouch === true || new Date() - this._lastTouch < 100
  },

  render() {
    return (
      <div onScroll={this._onScroll} onFocusCapture={this.onFocusCapture} onTouchStart={this.onTouchStart} onTouchEnd={this.onTouchEnd} className={this.props.className}>
        {this.props.children}
      </div>
    )
  },
})
