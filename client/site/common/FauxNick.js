import React from 'react'

import MessageText from '../../lib/ui/message-text'
import hueHash from '../../lib/hue-hash'


export default function FauxNick(props) {
  return <MessageText className="nick" onlyEmoji style={{background: 'hsl(' + hueHash.hue(props.nick) + ', 65%, 85%)'}} content={props.nick} />
}

FauxNick.propTypes = {
  nick: React.PropTypes.string,
}
