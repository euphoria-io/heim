import React from 'react'

import MessageText from '../../lib/ui/message-text'
import hueHash from '../../lib/hue-hash'


export default React.createClass({
  propTypes: {
    nick: React.PropTypes.string,
  },

  render() {
    return <MessageText className="nick" onlyEmoji style={{background: 'hsl(' + hueHash.hue(this.props.nick) + ', 65%, 85%)'}} content={this.props.nick} />
  },
})

