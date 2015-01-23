require('./support/setup')
var assert = require('assert')
var React = require('react/addons')
var TestUtils = React.addons.TestUtils


describe('<Message>', function() {
  var Tree = require('../lib/tree')
  var Message = require('../lib/ui/message')
  var testTree

  function renderMessage(content) {
    testTree = new Tree('time').reset([
      {
        'id': 'id1',
        'time': 123456,
        'sender': {
          'id': '32.64.96.128:12345',
          'name': 'tester',
        },
        'content': content
      }
    ])

    var message = TestUtils.renderIntoDocument(
      <Message tree={testTree} nodeId="id1" depth={0} />
    )

    return TestUtils.findRenderedDOMComponentWithClass(message, 'message')
  }

  it('automatically links urls', function() {
    var messageContent = renderMessage('http://google.com')
    assert.equal(messageContent.getDOMNode().innerHTML,
      '<a href="http://google.com" target="_blank" rel="noreferrer">google.com</a>')
  })

  it('truncates long urls', function() {
    var messageContent = renderMessage('http://google.com/abcdefghijklmnopqrstuvwxyz1234567890')
    assert.equal(messageContent.getDOMNode().innerHTML,
      '<a href="http://google.com/abcdefghijklmnopqrstuvwxyz1234567890" target="_blank" rel="noreferrer">google.com/abcdefghijklmnopqrstuvwxyz1..</a>')
  })

  it('doesn\'t linkify javascript:// links', function() {
    // note: jshint warns about javascript:// URLs
    var messageContent = renderMessage('Javascript://hello javascript://world')  // jshint ignore:line
    assert.equal(messageContent.getDOMNode().innerHTML,
      'Javascript://hello javascript://world')  // jshint ignore:line
  })
})
