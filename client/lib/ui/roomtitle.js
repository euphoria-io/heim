var React = require('react/addons')

var Bubble = require('./bubble')


module.exports = React.createClass({
  displayName: 'RoomTitle',

  mixins: [require('react-immutable-render-mixin')],

  showPrivacyInfo: function() {
    this.refs.privacyInfo.show()
  },

  render: function() {
    return (
      <span>
        <span className="room">
          <a className="name" href={'/room/' + this.props.name} onClick={ev => ev.preventDefault()}>&amp;{this.props.name}</a>
          {this.props.authType && <button className="private" onClick={this.showPrivacyInfo}>private</button>}
        </span>
        <Bubble ref="privacyInfo" className="small-text privacy-info" rightOffset={this.props.rightOffset}>
          {this.props.authType == 'passcode' && 'this room requires a passcode for entry.'}
        </Bubble>
      </span>
    )
  },
})
