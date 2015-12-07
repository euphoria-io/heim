import React from 'react'

import MessageText from '../../lib/ui/MessageText'
import hueHash from '../../lib/hueHash'


export default function FauxNick(props) {
  return <MessageText className="nick" onlyEmoji style={{background: 'hsl(' + hueHash.hue(props.nick) + ', 65%, 85%)'}} content={props.nick} />
}

FauxNick.propTypes = {
  nick: React.PropTypes.string,
}
