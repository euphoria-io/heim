var React = require('react/addons')

var FastButton = require('./fastbutton')
var Bubble = require('./bubble')


module.exports = React.createClass({
  displayName: 'RoomTitle',

  mixins: [require('react-immutable-render-mixin')],

  showPrivacyInfo: function() {
    this.refs.privacyInfo.show()
  },

  render: function() {
    var privacyLevel
    var privacyMsg
    switch (this.props.authType) {
      case 'passcode':
        privacyLevel = 'private'
        privacyMsg = 'this room requires a passcode for entry'
        break
      default:
        privacyLevel = 'public'
        privacyMsg = 'anyone with a link can join this room'
    }

    return (
      <span className="room-container">
        <span className="room">
          <a className="name" href={'/room/' + this.props.name} onClick={ev => ev.preventDefault()}>&amp;{this.props.name}</a>
          <FastButton fastTouch className={'privacy-level ' + privacyLevel} onClick={this.showPrivacyInfo}>{privacyLevel}</FastButton>
        </span>
        <Bubble ref="privacyInfo" className="small-text privacy-info" rightOffset={this.props.rightOffset}>
          {privacyMsg}
        </Bubble>
      </span>
    )
  },
})
