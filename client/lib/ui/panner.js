import _ from 'lodash'
import React from 'react'

import clamp from '../clamp'


export default React.createClass({
  displayName: 'Panner',

  propTypes: {
    threshold: React.PropTypes.number,
    smoothing: React.PropTypes.number,
    sensitivity: React.PropTypes.number,
    friction: React.PropTypes.number,
    snapPoints: React.PropTypes.objectOf(React.PropTypes.number),
    onMove: React.PropTypes.func,
    children: React.PropTypes.node,
  },

  getDefaultProps() {
    return {
      // minimum angle ratio accepted as a directional pan
      threshold: 2,

      // multiplier of past velocity in rolling velocity average
      smoothing: 1,

      // multiplier of boosted velocity when extrapolating intertial stopping point
      sensitivity: 3,

      // inverse of minimum movement speed
      friction: 0.02,
    }
  },

  componentDidMount() {
    this._drag = null
    this._x = null
    this._curIdx = 0
    this._animationFrame = null
  },

  componentWillUpdate(nextProps) {
    this._snapPoints = _.values(nextProps.snapPoints)
    this._snapPoints.sort()

    if (!_.isEqual(this.props.snapPoints, nextProps.snapPoints)) {
      const node = this.getDOMNode()
      node.style.transition = node.style.webkitTransition = 'none'
      node.style.transform = node.style.webkitTransform = 'translateX(' + this._clamp(this._getCurrentX()) + 'px)'
    }
  },

  onTouchStart(ev) {
    if (!this._drag || !this._drag.active) {
      const touch = ev.touches[0]
      const now = Date.now()
      this._drag = {
        id: touch.identifier,
        startX: touch.clientX,
        startY: touch.clientY,
        direction: null,
        initX: 0,
        x: null,
        vx: 0,
        lastX: 0,
        lastTime: now,
        active: true,
      }
      this.getDOMNode().style.willChange = 'transform'
    }
  },

  onTouchMove(ev) {
    if (!this._drag) {
      return
    }

    const touch = _.find(ev.touches, {identifier: this._drag.id})
    if (!touch) {
      return
    }

    if (this._drag.direction) {
      if (ev.cancelable) {
        ev.preventDefault()
      }
      this._drag.x = touch.clientX - this._drag.startX - this._drag.initX
      this._startAnimating()
      return
    }

    const deltaX = Math.abs(touch.clientX - this._drag.startX)
    const deltaY = Math.abs(touch.clientY - this._drag.startY)
    if (deltaX / deltaY >= this.props.threshold) {
      this._drag.direction = 'horizontal'
      this._drag.initX = deltaY
    } else if (deltaX < this.props.threshold / 2 || deltaX > this.props.threshold || deltaY > this.props.threshold) {
      // cancel drag if vertically panning or outside threshold
      this._drag = null
      this.getDOMNode().style.willChange = ''
    }
  },

  onTouchEnd(ev) {
    if (!this._drag) {
      return
    }

    if (ev.touches.length) {
      const touch = ev.touches[0]
      _.assign(this._drag, {
        id: touch.identifier,
        startX: touch.clientX,
        startY: touch.clientY,
        initX: -this._drag.x,
      })
    } else {
      if (this._drag.direction) {
        this._drag.active = false
      } else {
        this._drag = null
      }
    }
  },

  onTouchCancel(ev) {
    this.onTouchEnd(ev)
  },

  _intercept(name) {
    return (ev) => {
      this[name](ev)
      if (this.props[name]) {
        this.props[name](ev)
      }
    }
  },

  _startAnimating() {
    if (!this._animationFrame) {
      this._animationFrame = uiwindow.requestAnimationFrame(() => this._updateFrame())
    }
  },

  _getCurrentX() {
    const node = this.getDOMNode()
    return node.getBoundingClientRect().left - node.offsetLeft
  },

  _clamp(x) {
    return Math.round(clamp(this._snapPoints[0], x, _.last(this._snapPoints)))
  },

  flingTo(pointSpec, flingVx) {
    let point = pointSpec
    if (_.isString(point)) {
      point = this.props.snapPoints[point]
    }

    const node = this.getDOMNode()
    const x = this._getCurrentX()

    let vx = flingVx || 0
    const minV = Math.sqrt(2 * this.props.friction * Math.abs(point - x))
    const distance = point - x
    const targetDirection = Math.sign(distance)
    const movingFastEnough = Math.abs(vx) > minV
    const movingRightDirection = targetDirection === Math.sign(vx)
    // clamp velocity to the minimum required to bring us to the target point
    vx = (!movingFastEnough || !movingRightDirection) ? targetDirection * minV : vx
    const duration = Math.abs(vx) / this.props.friction

    // use parabolic easing curve (see http://stackoverflow.com/a/16883488)
    node.style.transition = node.style.webkitTransition = 'all ' + (duration / 1000) + 's cubic-bezier(0.33333, 0.66667, 0.66667, 1)'
    node.style.transform = node.style.webkitTransform = 'translateX(' + point + 'px)'
    _.identity(node.offsetHeight)  // reflow so transition starts immediately

    if (this.props.onMove) {
      this.props.onMove(_.findKey(this.props.snapPoints, snap => snap === point), duration)
    }
  },

  _updateFrame() {
    const now = Date.now()

    if (this._drag) {
      const node = this.getDOMNode()

      if (this._x === null) {
        this._x = node.getBoundingClientRect().left - node.offsetLeft
      }

      const dt = now - this._drag.lastTime
      this._drag.lastTime = now

      const deltaX = this._drag.x - this._drag.lastX
      this._x = this._clamp(this._x + deltaX)
      const vx = deltaX / dt
      this._drag.vx = (vx + this.props.smoothing * this._drag.vx) / (1 + this.props.smoothing)

      // iOS Safari requires webkit prefixes :(
      node.style.transition = node.style.webkitTransition = 'none'
      node.style.transform = node.style.webkitTransform = 'translateX(' + this._x + 'px)'

      if (this._drag.active) {
        this._drag.lastX = this._drag.x
      } else {
        if (this._drag.direction) {
          // extrapolate final stopping point based on position and velocity
          let stopPoint
          if (this._drag.vx) {
            const vxDirection = Math.sign(this._drag.vx)
            const vxWeighted = vxDirection * Math.pow(Math.abs(this._drag.vx), 0.25) * this.props.sensitivity
            const vDragFriction = -vxDirection * this.props.friction
            const stopTime = -vxWeighted / vDragFriction
            stopPoint = 0.5 * vDragFriction * stopTime * stopTime + vxWeighted * stopTime + this._x
          } else {
            stopPoint = this._x
          }

          const point = _.min(this._snapPoints, snapPoint => Math.abs(snapPoint - stopPoint))
          this.flingTo(point, this._drag.vx)
        }
        this._drag = null
      }
    }

    if (this._drag) {
      this._animationFrame = uiwindow.requestAnimationFrame(() => this._updateFrame())
    } else {
      this.getDOMNode().style.willChange = ''
      this._animationFrame = null
      this._x = null
    }
  },

  render() {
    return (
      <div {...this.props} onTouchStart={this._intercept('onTouchStart')} onTouchMove={this._intercept('onTouchMove')} onTouchEnd={this._intercept('onTouchEnd')} onTouchCancel={this._intercept('onTouchCancel')}>
        {this.props.children}
      </div>
    )
  },
})
