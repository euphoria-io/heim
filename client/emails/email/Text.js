import React from 'react'


export default function Text(props) {
  return (
    <span style={{
      fontFamily: props.fontFamily,
      fontSize: props.fontSize,
      fontWeight: props.fontWeight,
      lineHeight: props.lineHeight !== null ? props.lineHeight : props.fontSize,
      color: props.color,
      ...props.style,
    }}>{props.children}</span>
  )
}

Text.propTypes = {
  fontFamily: React.PropTypes.string,
  fontSize: React.PropTypes.number,
  fontWeight: React.PropTypes.string,
  lineHeight: React.PropTypes.number,
  color: React.PropTypes.string,
  style: React.PropTypes.object,
  children: React.PropTypes.node,
}

Text.defaultProps = {
  fontFamily: 'sans-serif',
  fontSize: 14,
  color: '#000',
}
