import React from 'react'

import { Box, Item, A, Image } from 'react-html-email'


export default function TopBubbleBox(props) {
  return (
    <Item align="center">
      <A href="{{.SiteURL}}">
        <Image src={'{{.File `' + props.logo + '`}}'} width={67} height={90} />
      </A>
      <Box width="600" cellPadding={2} bgcolor="white" style={{
        borderBottom: '3px solid #ccc',
        borderRadius: '10px',
        padding: props.padding,
      }}>
        {props.children}
      </Box>
    </Item>
  )
}

TopBubbleBox.propTypes = {
  logo: React.PropTypes.string.isRequired,
  padding: React.PropTypes.number,
  children: React.PropTypes.node,
}

TopBubbleBox.defaultProps = {
  padding: 7,
}
