var assert = require('assert')
var Immutable = require('immutable')


describe('mention', function() {
  var mention = require('../lib/mention')

  describe('containsSubseq', function() {
    function assertContains(n, p, yes) {
      assert.equal(mention.containsSubseq(n, p), yes)
    }

    it('handles empty and exact matches', function() {
      assertContains('', '', true)
      assertContains('name', '', true)
      assertContains('', 'cruft', false)
      assertContains('name', 'name', true)
    })
    it('rejects extra on end of partial', function() {
      assertContains('name', 'namex', false)
    })
    it('finds standard cases', function() {
      assertContains('name', 'nm', true)
      assertContains('name', 'e', true)
    })
    // This is a possible index bug.
    it('rejects extra chars going off the end', function() {
      assertContains('name', 'es', false)
    })
    it('respects order', function() {
      assertContains('name', 'eman', false)
    })
    it('respects count (always advance)', function() {
      assertContains('name', 'nnm', false)
    })
  })

  describe('scoreMatch', function() {
    
    it('scores entirely bad matches as null', function() {
      assert.deepEqual(mention.scoreMatch('', 'something'), null)
      assert.deepEqual(mention.scoreMatch('1', '2'), null)
    })
    it('empty prefix always matches', function() {
      assert.deepEqual(mention.scoreMatch('', ''), [-31, 0])
      assert.deepEqual(mention.scoreMatch('something', ''), [-31, 0])
    })
    it('has expected score levels for a single name', function() {
      // Exact match scores same as prefix for now.
      assert.deepEqual(mention.scoreMatch('TimMc', 'TimMc'), [-31, 0])
      assert.deepEqual(mention.scoreMatch('TimMc', 'timmc'), [-30, 0])
      assert.deepEqual(mention.scoreMatch('TimMc', 'Tim'), [-31, 0])
      assert.deepEqual(mention.scoreMatch('TimMc', 'tim'), [-30, 0])
      assert.deepEqual(mention.scoreMatch('TimMc', 'imM'), [-21, 1])
      assert.deepEqual(mention.scoreMatch('TimMc', 'imm'), [-20, 1])
      assert.deepEqual(mention.scoreMatch('TimMc', 'TM'), [-11, 0])
      assert.deepEqual(mention.scoreMatch('TimMc', 'tm'), [-10, 0])
      assert.deepEqual(mention.scoreMatch('TimMc', 'no'), null)
    })
  })

  describe('rankCompletions', function() {
    function assertRanking(names, part, outNames) {
      var actual = mention
        .rankCompletions(Immutable.Seq(names), part)
        .toArray()
      assert.deepEqual(actual, outNames)
    }
    var users = ['chromakode', 'logan', 'mac', 'Max 2', 'TimMc']

    it('removes entries that don\'t match at all', function() {
      assertRanking(users, 'g', ['logan'])
    })
    it('puts prefix over infix and tie-breaks with case', function() {
      // Note that this demonstrates that a case-insensitive prefix
      // match is superior to a case-sensitive infix match; otherwise
      // we could treat prefix as a high-scoring infix.
      assertRanking(users, 'M', ['Max2', 'mac', 'TimMc', 'chromakode'])
    })
    it('ranks subseqs less than infix ci', function() {
      assertRanking(users, 'mc', ['TimMc', 'mac'])
    })
    it('sorts asciibetically as tie-breaker, caps first', function() {
      assertRanking(['ax', 'Ax', 'Zx', 'zx'], 'x', ['Ax', 'ax', 'Zx', 'zx'])
    })
    it('ranks earlier infix over later', function() {
      // use a vs b to check for interference from fallback sorting
      assertRanking(['aXX', 'bbXX'], 'XX', ['aXX', 'bbXX'])
      assertRanking(['aaXX', 'bXX'], 'XX', ['bXX', 'aaXX'])
    })
    it('strips spaces from names', function() {
      assertRanking(users, 'x2', ['Max2'])
    })
  })
})
