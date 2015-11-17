import React from 'react'


export default function HeimAttachPoint(props) {
  return <div id={props.id} data-context="{{.Data}}" />
}

HeimAttachPoint.propTypes = {
  id: React.PropTypes.string,
}
