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
  <Email title="you're invited to join euphoria" bgcolor="#f0f0f0" cellSpacing="30">
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
          <Text {...textDefaults} fontSize={18}>hi! @chromakode invited you to a [public/private] chat room:</Text>
        </Item>
        <Item align="center">
          <Link href="https://euphoria.io/room/space">
            <Text {...textDefaults} fontSize={32} color={null}>&space</Text>
          </Link>
        </Item>
      </Box>
    </Item>
    <Item>
      <Box cellSpacing={20} width="100%" bgcolor="white" style={{
        borderBottom: '3px solid #ccc',
      }}>
        <Item align="center">
          <Text {...textDefaults} color="#7d7d7d">a note from @chromakode:</Text>
        </Item>
        <Item>
          <Text {...textDefaults}>hey intortus, is your refrigerator running?</Text>
        </Item>
      </Box>
    </Item>
    <Item>
      <Box cellSpacing={20} width="100%" bgcolor="white" style={{borderBottom: '3px solid #ccc'}}>
        <Item>
          <Text {...textDefaults}><Link href="https://euphoria.io/room/space">&space</Link> is hosted on <Link {...textDefaults} href="https://euphoria.io">euphoria</Link>, a free online discussion space. you don't have to sign up to chat &ndash; just click the link, choose a name, and you'll be chatting with @chromakode in moments.</Text>
        </Item>
      </Box>
    </Item>
    <Item style={{paddingLeft: '20px'}}>
      <Text {...textDefaults} fontSize={13} color="#7d7d7d">this message was sent to <Link {...textDefaults} textDecoration="none" href="mailto:c@chromakode.com">c@chromakode.com</Link> because a user on <Link {...textDefaults} textDecoration="none" href="https://euphoria.io">euphoria.io</Link> requested that we send you an invite. would you like to change your <Link {...textDefaults} href="https://euphoria.io">email notification preferences</Link>?</Text>
    </Item>
  </Email>
)
