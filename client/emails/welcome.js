var React = require('react')

var email = require('./email')
var Email = email.Email
var Box = email.Box
var Item = email.Item
var Text = email.Text
var Link = email.Link
var Image = email.Image


var textDefaults = {
  fontFamily: 'Verdana, sans-serif',
  fontSize: '16px',
  color: '#4d4d4d',
}

module.exports = (
  <Email title="welcome to euphoria!" bgcolor="#f0f0f0" cellSpacing="30">
    <Item align="center">
      <Link href="https://euphoria.io">
        <Image src="logo.png" width={67} height={90} />
      </Link>
      <Box width="600" cellPadding="2" bgcolor="white" style={{
        borderBottom: '3px solid #ccc',
        borderRadius: '10px',
        padding: '7px',
      }}>
        <Item align="center">
          <Text {...textDefaults} fontSize={52}>hi!</Text>
        </Item>
        <Item align="center">
          <Text {...textDefaults} fontSize={18} color="#9f9f9f">welcome to euphoria :)</Text>
        </Item>
      </Box>
    </Item>
    <Item>
      <Box cellPadding={20} bgcolor="white" style={{borderBottom: '3px solid #ccc'}}>
        <Item align="center">
          <Text {...textDefaults}>your account is almost ready:</Text>
        </Item>
        <Item align="center" cellPadding={24}>
          <Link color="white" textDecoration="none" href="https://euphora.io/wherever" style={{
            background: '#80c080',
            padding: '24px 30px',
            borderRadius: '4px',
          }}>
            <Text {...textDefaults} fontSize={24} fontWeight="bold" color="white">verify your email address</Text>
          </Link>
        </Item>
        <Item>
          <Text {...textDefaults}>we hope you have a wonderful time on euphoria. if you have any questions or comments, feel free to <Link {...textDefaults} href="mailto:hi@euphoria.io">contact us</Link>.</Text>
        </Item>
      </Box>
    </Item>
    <Item style={{paddingLeft: '20px'}}>
      <Text {...textDefaults} fontSize={13} color="#7d7d7d">this message was sent to <Link {...textDefaults} textDecoration="none" href="mailto:c@chromakode.com">c@chromakode.com</Link> because someone signed up for an account on <Link {...textDefaults} textDecoration="none" href="https://euphoria.io">euphoria.io</Link> with this email address. if you did not request this email, feel free to disregard.</Text>
    </Item>
  </Email>
)
