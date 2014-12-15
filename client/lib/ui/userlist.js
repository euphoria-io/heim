var _ = require('lodash')
var React = require('react')
var moment = require('moment')
var autolinker = require('autolinker')


module.exports = {}

module.exports = React.createClass({
  render: function() {
    return (
      <div className="user-list" {...this.props}>
        {_.map(this.props.users, function(user, idx) {
          return <div key={user.id} className="line"><span className="nick" style={{background: 'hsl(' + this.props.hues[user.name] + ', 65%, 85%)'}}>{user.name}</span></div>
        }, this)}
      </div>
    )
  },
})
