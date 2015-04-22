require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')
var Immutable = require('immutable')

var Tree = require('../lib/tree')

describe('Tree', function() {
  function debugMap(node, children, depth) {
    return [node.get('id'), depth, children.toArray()]
  }

  function checkMapOnlyRoot(tree) {
    assert.deepEqual(tree.mapDFS(debugMap), ['__root', 0, []])
  }

  function expectEmit(tree, ids) {
    sinon.assert.callCount(tree.changes.emit, ids.length + 1)
    Immutable.Seq(ids).forEach(id => {
      sinon.assert.calledWithExactly(tree.changes.emit, id, tree.get(id))
    })
    sinon.assert.calledWithExactly(tree.changes.emit, '__all', ids)
  }

  describe('a new empty tree', function() {
    var tree = new Tree()

    it('should have size 0', function() {
      assert.equal(tree.size, 0)
    })

    it('should only visit root in traversal', function() {
      checkMapOnlyRoot(tree)
    })
  })

  describe('initialized with an array of entries', function() {
    var entries = [
      {id: '1', value: 'hello', time: 5},
      {id: '2', parent: '1', value: 'world', time: 5},
    ]
    var tree
    var updateFunc
    var prevRoot

    beforeEach(function() {
      updateFunc = sinon.spy()
      tree = new Tree('time', updateFunc)
      prevRoot = tree.get('__root')
      sinon.stub(tree.changes, 'emit')
      tree.reset(entries)
    })

    afterEach(function() {
      tree.changes.emit.restore()
    })

    it('should have correct size', function() {
      assert.equal(tree.size, 2)
    })

    it('should nest nodes with parents', function() {
      assert(tree.get('1').get('children').contains('2'))
    })

    it('should call the updateFunc with the old changed nodes and root', function() {
      sinon.assert.calledWithExactly(tree.updateFunc, {
        '__root': prevRoot,
        '1': true,
        '2': true,
      }, sinon.match.func)
    })

    it('should trigger a change event on the new nodes and root', function() {
      expectEmit(tree, ['__root', '1', '2'])
    })

    it('should visit all nodes in a map traversal', function() {
      var visited = tree.mapDFS(debugMap)
      assert.deepEqual(visited, [
        '__root', 0, [
          ['1', 1, [
            ['2', 2, []]
          ]]
        ]
      ])
    })

    describe('after adding a node', function() {
      var prev1

      beforeEach(function() {
        tree.changes.emit.reset()
        updateFunc.reset()
        prev1 = tree.get('1')
        tree.add({id: '3', parent: '1', value: 'yo', time: 7})
      })

      it('should contain the new node', function() {
        assert(tree.get('1').get('children').contains('3'))
      })

      it('should update the size', function() {
        assert.equal(tree.size, 3)
      })

      it('should call the updateFunc with the old changed nodes', function() {
        sinon.assert.calledWithExactly(tree.updateFunc, {
          '1': prev1,
          '3': true,
        }, sinon.match.func)
      })

      it('should trigger a change event on the new node and parent', function() {
        expectEmit(tree, ['1', '3'])
      })

      it('the new node should be last', function() {
        assert.equal(tree.last(), tree.get('3'))
      })

      it('should visit all nodes in a map traversal', function() {
        var visited = tree.mapDFS(debugMap)
        assert.deepEqual(visited, [
          '__root', 0, [
            ['1', 1, [
              ['2', 2, []],
              ['3', 2, []],
            ]]
          ]
        ])
      })
    })

    describe('after adding multiple nodes', function() {
      var prevRoot
      var prev1

      var entries = [
        {id: '2', parent: '1', value: 'world', time: 5},
        {id: '0', value: 'first!', time: 0},
        {id: '3', parent: '1', value: 'local first!', time: 1},
        {id: '9', value: 'last', time: 9},
      ]

      beforeEach(function() {
        tree.changes.emit.reset()
        updateFunc.reset()
        prevRoot = tree.get('__root')
        prev1 = tree.get('1')
        tree.add(entries)
      })

      function check() {
        it('the size should be correct', function() {
          assert.equal(tree.size, 5)
        })

        it('the node with highest time should be last', function() {
          assert.equal(tree.last(), tree.get('9'))
          assert.equal(tree._lastValue, 9)
        })

        it('should visit all nodes in a map traversal (in the right order)', function() {
          var visited = tree.mapDFS(debugMap)
          assert.deepEqual(visited, [
            '__root', 0, [
              ['0', 1, []],
              ['1', 1, [
                ['3', 2, []],
                ['2', 2, []],
              ]],
              ['9', 1, []],
            ]
          ])
        })

        it('should visit all children of node 1 in a child traversal', function() {
          var nodes = Immutable.Seq(tree.iterChildrenOf('1'))
          var expectedNodes = Immutable.Seq(['3', '2']).map(id => tree.get(id))
          assert.deepEqual(nodes.toJS(), expectedNodes.toJS())
        })

        it('should visit all ancestors of node 3 in an ancestor traversal', function() {
          var nodes = Immutable.Seq(tree.iterAncestorsOf('3'))
          var expectedNodes = Immutable.Seq(['1', '__root']).map(id => tree.get(id))
          assert.deepEqual(nodes.toJS(), expectedNodes.toJS())
        })
      }

      it('should call the updateFunc with the old changed nodes', function() {
        sinon.assert.calledWithExactly(tree.updateFunc, {
          '__root': prevRoot,
          '1': prev1,
          '0': true,
          '3': true,
          '9': true,
        }, sinon.match.func)
      })

      it('should only trigger a change event for new nodes and the parents of new nodes', function() {
        expectEmit(tree, ['__root', '0', '1', '3', '9'])
      })

      describe('after re-adding the same nodes', function() {
        beforeEach(function() {
          tree.changes.emit.reset()
          updateFunc.reset()
          tree.add(entries)
        })

        describe('should not change', function() {
          check(tree)
        })

        it('should not call the updateFunc', function() {
          sinon.assert.notCalled(tree.updateFunc)
        })

        it('should not trigger a change event', function() {
          sinon.assert.notCalled(tree.changes.emit)
        })
      })

      describe('after re-adding a node with an inconsequential change to sort prop', function() {
        var prev3

        beforeEach(function() {
          tree.changes.emit.reset()
          updateFunc.reset()
          prev3 = tree.get('3')
          tree.add({id: '3', parent: '1', time: 2})
        })

        it('should call the updateFunc with only the old changed node (not the parent)', function() {
          sinon.assert.calledWithExactly(tree.updateFunc, {
            '3': prev3,
          }, sinon.match.func)
        })

        it('should trigger a change event on only the node (not the parent)', function() {
          expectEmit(tree, ['3'])
        })
      })

      describe('cloning the tree', function() {
        beforeEach(function() {
          tree = tree.clone()
          sinon.stub(tree.changes, 'emit')
        })

        describe('should produce a tree which', function() {
          check()
        })
      })
    })

    describe('after adding a node with a missing parent', function() {
      beforeEach(function() {
        tree.changes.emit.reset()
        updateFunc.reset()
        tree.add({id: '3', parent: 'wtf', value: 'yo', time: 7})
      })

      it('should contain the new node', function() {
        assert(tree.get('3'))
      })

      it('should contain an unreachable parent for the new node', function() {
        var parent = tree.get('wtf')
        assert(parent.get('children').contains('3'))
        assert(!parent.has('parent'))
      })

      it('should update the size', function() {
        assert.equal(tree.size, 4)
      })

      it('should call the updateFunc with the new node', function() {
        sinon.assert.calledWithExactly(tree.updateFunc, {
          'wtf': true,
          '3': true,
        }, sinon.match.func)
      })

      describe('after adding the missing parent', function() {
        var prev1
        var prevWtf

        beforeEach(function() {
          tree.changes.emit.reset()
          updateFunc.reset()
          prev1 = tree.get('1')
          prevWtf = tree.get('wtf')
          tree.add({id: 'wtf', parent: '1', value: 'j0', time: 6})
        })

        it('should update the size', function() {
          assert.equal(tree.size, 4)
        })

        it('should call the updateFunc with the new node', function() {
          sinon.assert.calledWithExactly(tree.updateFunc, {
            '1': prev1,
            'wtf': prevWtf,
          }, sinon.match.func)
        })

        it('should trigger a change event on the child and parent', function() {
          expectEmit(tree, ['1', 'wtf'])
        })

        it('should visit all nodes in a map traversal', function() {
          var visited = tree.mapDFS(debugMap)
          assert.deepEqual(visited, [
            '__root', 0, [
              ['1', 1, [
                ['2', 2, []],
                ['wtf', 2, [
                  ['3', 3, []]
                ]],
              ]]
            ]
          ])
        })
      })
    })

    describe('after adding a node with an existing other parent (reparenting)', function() {
      var prevRoot
      var prev1
      var prev2

      beforeEach(function() {
        tree.changes.emit.reset()
        updateFunc.reset()
        prevRoot = tree.get('__root')
        prev1 = tree.get('1')
        prev2 = tree.get('2')
        tree.add({id: '2', parent: '__root', value: 'ayyyy', time: 7})
      })

      it('should contain the node', function() {
        assert(tree.get('2'))
      })

      it('should change the parent of the node', function() {
        assert(tree.get('2').get('parent') == '__root')
      })

      it('should not change the size', function() {
        assert.equal(tree.size, 2)
      })

      it('should call the updateFunc with the old changed nodes', function() {
        sinon.assert.calledWithExactly(tree.updateFunc, {
          '__root': prevRoot,
          '1': prev1,
          '2': prev2,
        }, sinon.match.func)
      })

      it('should trigger a change event for the old parent, node, and new parent', function() {
        expectEmit(tree, ['__root', '1', '2'])
      })

      it('should visit all nodes in a map traversal', function() {
        var visited = tree.mapDFS(debugMap)
        assert.deepEqual(visited, [
          '__root', 0, [
            ['1', 1, []],
            ['2', 1, []],
          ]
        ])
      })
    })

    describe('after adding a node with no parent ("shadow")', function() {
      beforeEach(function() {
        tree.changes.emit.reset()
        updateFunc.reset()
        tree.add({id: 'shadow', parent: null, value: 'boo'})
      })

      it('should contain the node with no parent', function() {
        assert(tree.get('shadow'))
        assert.equal(tree.get('shadow').get('parent'), null)
      })

      it('should change the size', function() {
        assert.equal(tree.size, 3)
      })

      it('should call the updateFunc with shadow node', function() {
        sinon.assert.calledWithExactly(tree.updateFunc, {
          'shadow': true,
        }, sinon.match.func)
      })

      it('should trigger a change event for the old parent, node, and new parent', function() {
        expectEmit(tree, ['shadow'])
      })

      describe('after adding the node with a parent', function() {
        var prevRoot
        var prevShadow

        beforeEach(function() {
          tree.changes.emit.reset()
          updateFunc.reset()
          prevRoot = tree.get('__root')
          prevShadow = tree.get('shadow')
          tree.add({id: 'shadow', parent: '__root', time: 1})
        })

        it('should contain the node with shadow properties merged in', function() {
          assert(tree.get('shadow'))
          assert.deepEqual(tree.get('shadow').toJS(), {
            id: 'shadow',
            parent: '__root',
            children: [],
            time: 1,
            value: 'boo',
          })
        })

        it('should have the correct size', function() {
          assert.equal(tree.size, 3)
        })

        it('should call the updateFunc with the shadow node and its parent', function() {
          sinon.assert.calledWithExactly(tree.updateFunc, {
            '__root': prevRoot,
            'shadow': prevShadow,
          }, sinon.match.func)
        })

        it('should trigger a change event for the old parent, node, and new parent', function() {
          expectEmit(tree, ['__root', 'shadow'])
        })
      })
    })

    describe('after merging an update to a node', function() {
      it('should update the node', function() {
        tree.mergeNodes('2', {value: 'dawg'})
        assert.equal(tree.get('2').get('value'), 'dawg')
      })

      it('should retain the same size', function() {
        tree.mergeNodes('2', {value: 'dawg'})
        assert.equal(tree.size, 2)
      })

      it('should call updateFunc and trigger a change event', function() {
        tree.updateFunc.reset()
        tree.changes.emit.reset()
        var prev2 = tree.get('2')
        tree.mergeNodes('2', {value: 'dawg'})
        sinon.assert.calledWithExactly(tree.updateFunc, {
          '2': prev2,
        }, sinon.match.func)
        expectEmit(tree, ['2'])
      })

      it('should not call updateFunc or trigger a change event if unchanged', function() {
        tree.updateFunc.reset()
        tree.changes.emit.reset()
        tree.mergeNodes('2', {value: 'world'})
        sinon.assert.notCalled(tree.updateFunc)
        sinon.assert.notCalled(tree.changes.emit)
      })
    })

    describe('after merging a bulk update to a node', function() {
      it('should update the node', function() {
        tree.mergeNodes(['1', '2'], {value: 'bulk'})
        assert.equal(tree.get('1').get('value'), 'bulk')
        assert.equal(tree.get('2').get('value'), 'bulk')
      })
    })

    describe('after resetting to empty', function() {
      beforeEach(function() {
        tree.changes.emit.reset()
        tree.reset()
      })

      it('should be empty', function() {
        assert.equal(tree.size, 0)
      })

      it('should only visit root in traversal', function() {
        checkMapOnlyRoot(tree)
      })

      it('should trigger a change event', function() {
        expectEmit(tree, ['__root'])
      })
    })
  })

  describe('initialized with an updateFunc', function() {
    var entries = [
      {id: '1', value: 'hello', time: 5},
      {id: '2', value: 'hi', time: 6},
    ]
    var tree

    function callUpdateNode(updateNode, newNode) {
      assert.equal(updateNode(newNode), newNode)
    }

    beforeEach(function() {
      tree = new Tree('time', function(oldNodes, updateNode) {
        Immutable.Seq(oldNodes)
          .forEach((oldNode, id) => {
            callUpdateNode(updateNode, tree.get(id).set('updated', true))
          })

        // test redundant node updates (emitting a change)
        var node2 = tree.get('2')
        callUpdateNode(updateNode, node2.set('alwaysSet', true))

        // test updating an unrelated (unchanged) node
        var rootNode = tree.get('__root')
        var newUpdateCount = rootNode.get('updateCount', 0) + 1
        callUpdateNode(updateNode, rootNode.set('updateCount', newUpdateCount))
      })
      sinon.stub(tree.changes, 'emit')
      tree.reset(entries)
    })

    afterEach(function() {
      tree.changes.emit.restore()
    })

    it('should have updated each node', function() {
      assert.equal(tree.get('__root').get('updated'), true)
      assert.equal(tree.get('1').get('updated'), true)
      assert.equal(tree.get('__root').get('updateCount'), 1)
    })

    it('should trigger a change event on the new nodes and root', function() {
      expectEmit(tree, ['__root', '1', '2'])
    })

    describe('after adding a node', function() {
      beforeEach(function() {
        tree.changes.emit.reset()
        tree.add({id: '3', parent: '1', value: 'yo', time: 7})
      })

      it('should have updated it and the updateCount', function() {
        assert.equal(tree.get('3').get('updated'), true)
        assert.equal(tree.get('__root').get('updateCount'), 2)
      })

      it('should trigger a change event on the new node and all updated nodes', function() {
        expectEmit(tree, ['1', '3', '__root'])
      })
    })
  })
})
