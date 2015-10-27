var support = require('./support/setup')
var assert = require('assert')
var React = require('react/addons')
var TestUtils = React.addons.TestUtils


describe('<MessageText>', function() {
  var MessageText = require('../lib/ui/message-text')

  support.fakeEnv({
    HEIM_PREFIX: '/test',
  })

  function renderMessageText(content) {
    return TestUtils.renderIntoDocument(
      <MessageText content={content} />
    )
  }

  it('automatically links urls', function() {
    var messageContent = renderMessageText('http://google.com')
    assert.equal(messageContent.getDOMNode().innerHTML,
      '<a href="http://google.com" target="_blank" rel="noreferrer">google.com</a>')
  })

  it('truncates long urls', function() {
    var messageContent = renderMessageText('http://google.com/abcdefghijklmnopqrstuvwxyz1234567890')
    assert.equal(messageContent.getDOMNode().innerHTML,
      '<a href="http://google.com/abcdefghijklmnopqrstuvwxyz1234567890" target="_blank" rel="noreferrer">google.com/abcdefghijklmnopqrstuvwxyz1..</a>')
  })

  it('linkifies &room references', function() {
    var messageContent = renderMessageText('hello &space! foo&bar &bar &baz')
    assert.equal(messageContent.getDOMNode().innerHTML,
      'hello <a href="/test/room/space/" target="_blank">&amp;space</a>! foo&amp;bar <a href="/test/room/bar/" target="_blank">&amp;bar</a> <a href="/test/room/baz/" target="_blank">&amp;baz</a>')
  })

  it('doesn\'t linkify javascript:// links', function() {
    // note: jshint warns about javascript:// URLs
    var messageContent = renderMessageText('Javascript://hello javascript://world')  // jshint ignore:line
    assert.equal(messageContent.getDOMNode().innerHTML,
      'Javascript://hello javascript://world')  // jshint ignore:line
  })

  it('processes emoji', function() {
    var messageContent = renderMessageText(':euphoria: \u25b6 \ud83d\udd14 \u2122 \u00a9')
    assert.equal(messageContent.getDOMNode().innerHTML,
      '<div class="emoji emoji-euphoria" title=":euphoria:">:euphoria:</div> <div class="emoji emoji-25b6" title=":arrow_forward:">\u25b6</div> <div class="emoji emoji-1f514" title=":bell:">\ud83d\udd14</div> \u2122 \u00a9')
  })
})
