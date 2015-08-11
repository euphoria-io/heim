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
      sensitivity: 3,

      // inverse of minimum movement speed
      friction: 0.02,
    }
  },

  componentDidMount: function() {
    this._drag = null
    this._x = null
    this._curIdx = 0
    this._animationFrame = null
  },

  componentWillUpdate: function(nextProps) {
    this._snapPoints = _.values(nextProps.snapPoints)
    this._snapPoints.sort()

    if (!_.isEqual(this.props.snapPoints, nextProps.snapPoints)) {
      var node = this.getDOMNode()
      node.style.transition = 'none'
      node.style.transform = 'translateX(' + this._clamp(this._getCurrentX()) + 'px)'
    }
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
      var now = Date.now()
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
      this._startAnimating()
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
      if (this._drag.direction) {
        this._drag.active = false
      } else {
        this._drag = null
      }
    }
  },

  onTouchCancel: function(ev) {
    this.onTouchEnd(ev)
  },

  _startAnimating: function() {
    if (!this._animationFrame) {
      this._animationFrame = uiwindow.requestAnimationFrame(() => this._updateFrame())
    }
  },

  _getCurrentX: function() {
    var node = this.getDOMNode()
    return node.getBoundingClientRect().left - node.offsetLeft
  },

  _clamp: function(x) {
    return Math.round(clamp(this._snapPoints[0], x, _.last(this._snapPoints)))
  },

  flingTo: function(point, vx) {
    if (_.isString(point)) {
      point = this.props.snapPoints[point]
    }

    var node = this.getDOMNode()
    var x = this._getCurrentX()

    vx = vx || 0
    var minV = Math.sqrt(2 * this.props.friction * Math.abs(point - x))
    var distance = point - x
    var targetDirection = Math.sign(distance)
    var movingFastEnough = Math.abs(vx) > minV
    var movingRightDirection = targetDirection == Math.sign(vx)
    // clamp velocity to the minimum required to bring us to the target point
    vx = (!movingFastEnough || !movingRightDirection) ? targetDirection * minV : vx
    var duration = Math.abs(vx) / this.props.friction

    // use parabolic easing curve (see http://stackoverflow.com/a/16883488)
    node.style.transition = 'transform ' + (duration / 1000) + 's cubic-bezier(0.33333, 0.66667, 0.66667, 1)'
    node.style.transform = 'translateX(' + point + 'px)'
    _.identity(node.offsetHeight)  // reflow so transition starts immediately

    if (this.props.onMove) {
      this.props.onMove(_.findKey(this.props.snapPoints, snap => snap == point), duration)
    }
  },

  _updateFrame: function() {
    var now = Date.now()

    if (this._drag) {
      var node = this.getDOMNode()

      if (this._x === null) {
        this._x = node.getBoundingClientRect().left - node.offsetLeft
      }

      var dt = now - this._drag.lastTime
      this._drag.lastTime = now

      var deltaX = this._drag.x - this._drag.lastX
      this._x = this._clamp(this._x + deltaX)
      var vx = deltaX / dt
      this._drag.vx = (vx + this.props.smoothing * this._drag.vx) / (1 + this.props.smoothing)

      node.style.transition = 'none'
      node.style.transform = 'translateX(' + this._x + 'px)'

      if (this._drag.active) {
        this._drag.lastX = this._drag.x
      } else {
        if (this._drag.direction) {
          // extrapolate final stopping point based on position and velocity
          var stopPoint
          if (this._drag.vx) {
            var vxDirection = Math.sign(this._drag.vx)
            var vxWeighted = vxDirection * Math.pow(Math.abs(this._drag.vx), 0.25) * this.props.sensitivity
            var vDragFriction = -vxDirection * this.props.friction
            var stopTime = -vxWeighted / vDragFriction
            stopPoint = 0.5 * vDragFriction * stopTime * stopTime + vxWeighted * stopTime + this._x
          } else {
            stopPoint = this._x
          }

          var point = _.min(this._snapPoints, point => Math.abs(point - stopPoint))
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

  render: function() {
    return (
      <div {...this.props} onTouchStart={this._intercept('onTouchStart')} onTouchMove={this._intercept('onTouchMove')} onTouchEnd={this._intercept('onTouchEnd')} onTouchCancel={this._intercept('onTouchCancel')}>
        {this.props.children}
      </div>
    )
  },
})
