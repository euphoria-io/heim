var _ = require('lodash')
var React = require('react')

var clamp = require('../clamp')
require('../math-sign-polyfill')


module.exports = React.createClass({
  displayName: 'Panner',

  getDefaultProps: function() {
    return {
      // minimum angle ratio accepted as a directional pan
      threshold: 2,

      // multiplier of past velocity in rolling velocity average
      smoothing: 1,

      // multiplier of boosted velocity when extrapolating intertial stopping point
      sensitivity: 2,

      // inverse of minimum movement speed
      friction: .01,
    }
  },

  componentDidMount: function() {
    this._drag = null
    this._motion = {x: 0, vx: 0, lastTime: 0}
    this._curIdx = 0
    this._animationFrame = null
  },

  _intercept: function(name) {
    return (ev) => {
      this[name](ev)
      if (this.props[name]) {
        this.props[name](ev)
      }
    }
  },

  onTouchStart: function(ev) {
    if (!this._drag || !this._drag.active) {
      var touch = ev.touches[0]
      this._drag = {
        id: touch.identifier,
        startX: touch.clientX,
        startY: touch.clientY,
        startTime: Date.now(),
        direction: null,
        initX: 0,
        x: null,
        lastX: 0,
        active: true,
      }
      this.getDOMNode().style.willChange = 'transform'
    }
  },

  onTouchMove: function(ev) {
    if (!this._drag) {
      return
    }

    var touch = _.find(ev.touches, {identifier: this._drag.id})
    if (!touch) {
      return
    }

    if (this._drag.direction) {
      if (ev.cancelable) {
        ev.preventDefault()
      }
      this._drag.x = touch.clientX - this._drag.startX - this._drag.initX
      if (!this._animationFrame) {
        this._animationFrame = uiwindow.requestAnimationFrame(() => this._updateFrame())
      }
      return
    }

    var deltaX = Math.abs(touch.clientX - this._drag.startX)
    var deltaY = Math.abs(touch.clientY - this._drag.startY)
    if (deltaX / deltaY >= this.props.threshold) {
      this._drag.direction = 'horizontal'
      this._drag.initX = deltaY
    } else if (deltaX < this.props.threshold / 2 || deltaX > this.props.threshold || deltaY > this.props.threshold) {
      // cancel drag if vertically panning or outside threshold
      this._drag = null
      this.getDOMNode().style.willChange = ''
    }
  },

  onTouchEnd: function(ev) {
    if (!this._drag) {
      return
    }

    if (ev.touches.length) {
      var touch = ev.touches[0]
      _.assign(this._drag, {
        id: touch.identifier,
        startX: touch.clientX,
        startY: touch.clientY,
        initX: -this._drag.x,
      })
    } else {
      this._drag.active = false
    }
  },

  onTouchCancel: function(ev) {
    this.onTouchEnd(ev)
  },

  _flingTo: function(point) {
    // clamp velocity to the minimum required to bring us to a snap
    var minV = Math.sqrt(2 * this.props.friction * Math.abs(point - this._motion.x))
    var targetDirection = Math.sign(point - this._motion.x)
    var movingFastEnough = Math.abs(this._motion.vx) > minV
    var movingRightDirection = targetDirection == Math.sign(this._motion.vx)
    if (!movingFastEnough || !movingRightDirection) {
      this._motion.vx = targetDirection * minV
    }
  },

  _updateFrame: function() {
    var now = Date.now()

    var dt = now - Math.max(this._drag && this._drag.startTime, this._motion.lastTime)
    this._motion.lastTime = now

    if (this._drag) {
      var deltaX = this._drag.x - this._drag.lastX
      this._motion.x += deltaX
      var vx = deltaX / dt
      this._motion.vx = (vx + this.props.smoothing * this._motion.vx) / (1 + this.props.smoothing)

      if (this._drag.active) {
        this._drag.lastX = this._drag.x
      } else {
        // extrapolate final stopping point based on position and velocity
        var stopPoint
        if (this._motion.vx) {
          var vxDirection = Math.sign(this._motion.vx)
          var vxWeighted = vxDirection * Math.pow(Math.abs(this._motion.vx), .25) * this.props.sensitivity
          var vFriction = -vxDirection * this.props.friction
          var stopTime = -vxWeighted / vFriction
          stopPoint = .5 * vFriction * Math.pow(stopTime, 2) + vxWeighted * stopTime + this._motion.x
        } else {
          stopPoint = this._motion.x
        }

        var point = _.min(this.props.snapPoints, (point) => Math.abs(point - stopPoint))
        this._flingTo(point)

        this._drag = null
      }
    }

    if (!this._drag) {
      var vFriction = -Math.sign(this._motion.vx) * this.props.friction
      this._motion.x += .5 * vFriction * Math.pow(dt, 2) + this._motion.vx * dt
      this._motion.vx = Math.sign(this._motion.vx) * Math.max(0, Math.abs(this._motion.vx) - this.props.friction * dt)
    }

    var clamped = clamp(this.props.snapPoints[0], this._motion.x, _.last(this.props.snapPoints))
    if (this._motion.x != clamped) {
      console.log('clamp', clamped)
      this._motion.x = clamped
      this._motion.vx = 0
    }

    this.pan(this._motion)

    if (this._drag || this._motion.vx != 0) {
      this._animationFrame = uiwindow.requestAnimationFrame(() => this._updateFrame())
    } else {
      console.log(this._motion.x)
      this._animationFrame = null
      this._motion.lastTime = null
      this.pan(null)
    }
  },

  pan: function(motion) {
    var node = this.getDOMNode()

    if (motion) {
      var offset = motion.x
      node.style.transform = 'translateX(' + offset + 'px)'
    } else {
      this.getDOMNode().style.willChange = ''
    }
  },

  render: function() {
    return (
      <div {...this.props} onTouchStart={this._intercept('onTouchStart')} onTouchMove={this._intercept('onTouchMove')} onTouchEnd={this._intercept('onTouchEnd')} onTouchCancel={this._intercept('onTouchCancel')}>
        {this.props.children}
      </div>
    )
  },
})
