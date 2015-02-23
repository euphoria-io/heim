var React = require('react/addons')

var Bubble = require('./bubble')


module.exports = React.createClass({
  displayName: 'PrivacyBubble',

  mixins: [require('react-immutable-render-mixin')],

  show: function() {
    this.refs.bubble.show()
  },

  render: function() {
    return (
      <Bubble ref="bubble" className="small-text privacy-info" {...this.props}>
        {this.props.authType == 'passcode' && 'this room requires a passcode for entry.'}
      </Bubble>
    )
  },
})
