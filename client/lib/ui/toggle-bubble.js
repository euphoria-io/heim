var _ = require('lodash')
var React = require('react/addons')

var Bubble = require('./bubble')


module.exports = React.createClass({
  displayName: 'ToggleBubble',

  mixins: [require('react-immutable-render-mixin')],

  componentWillMount: function() {
    // queue cancelable hide so that if the click triggers a show, we don't
    // hide and then immediately reshow.
    this._hide = _.debounce(this.hide, 0)
  },

  getInitialState: function() {
    return {visible: false}
  },

  show: function() {
    this.setState({visible: true})
    this._hide.cancel()
  },

  hide: function() {
    this.setState({visible: false})
  },

  toggle: function() {
    if (this.state.visible) {
      this._hide()
    } else {
      this.show()
    }
  },

  render: function() {
    return (
      <Bubble {...this.props} visible={this.state.visible} onDismiss={this._hide}>
        {this.props.children}
      </Bubble>
    )
  },
})
