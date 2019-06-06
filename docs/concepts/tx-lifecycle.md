# Transaction Lifecycle


## Prerequisite Reading
* [High-level overview of the architecture of an SDK application](https://github.com/cosmos/cosmos-sdk/docs/intro/sdk-app-architecture.md)
* Baseapp concept doc (not written yet)

## Synopsis
1. **Definition and Creation:** Transactions are comprised of `Msg`s specified by the developer.
2. **Addition to Mempool:** Nodes that receive transactions validate them first by running stateless and stateful checks on a copy of the internal state. Approved transactions are kept in the node's Mempool pending inclusion in the next block.
3. **Consensus:** The proposer creates a block, finalizing the transactions and their order for this round. Validators run Tendermint BFT Consensus to come to agreement on a block.
4. **State Changes:** Transactions are delivered, creating state changes and expending gas. To commit, internal state is updated and all copies are reset; the new state root is returned as proof. 


```
		     CheckTxState		    DeliverTxState
		-----------------------		-----------------------
		| CheckTxState(t)(0)  |         | DeliverTxState(t)(0)|
		-----------------------		|                     |
CheckTx()	          |			|                     |
			  v			|                     |
		-----------------------		|                     |
		| CheckTxState(t)(1)  |         |                     |
		-----------------------		|                     |
CheckTx()	          |			|                     |
			  v			|                     |
		-----------------------		|                     |
		| CheckTxState(t)(2)  |         |   		      |
		-----------------------		|                     |
CheckTx()	          |			|                     |
			  v			|                     |
		-----------------------		|                     |
		| CheckTxState(t)(3)  |         -----------------------
DeliverTx()	|   		      |		           |          
		|   		      |		           v          
		|   		      |		-----------------------
		|   		      |		| DeliverTxState(t)(1)|
		|   		      |		-----------------------
DeliverTx()	|   		      |	                   |
		|   		      |			   v
		|   		      |		-----------------------
		|   		      |	      	| DeliverTxState(t)(2)|
		|   		      |		-----------------------
DeliverTx()	|   		      |	                   |
		|   		      |			   v
		|   		      |		-----------------------
		|   		      |	      	| DeliverTxState(t)(3)|
		-----------------------		-----------------------
Commit()		  |				   |
			  v				   v
		-----------------------		-----------------------
		| CheckTxState(t+1)   |         | DeliverTxState(t+1) |
		-----------------------		|                     |
		          |			|                     |
			  v			|                     |			  
			  .				   .
			  .				   . 
			  .				   .
```

## Definition and Creation
Transactions are comprised of one or multiple **Messages** (link concept doc) and trigger state changes. The developer defines the specific messages to describe possible actions for the application by implementing the [`Msg`]() interface. 

The user performs an action to change the state, thereby creating a transaction, and provides a value `GasWanted` indicating the maximum amount of gas he is willing to spend to make this action go through. The node from which this transaction originates broadcasts it to its peers. 

## Addition to Mempool
Each full node that receives a transaction performs local sanity checks to filter out and discard invalid transactions before they get included in a block. The first check is to unwrap the transaction into its message(s) and run each `validateBasic` function, which is simply a stateless sanity check (e.g. nonnegative numbers, nil strings, empty addresses). A stateful ABCI validation function, `CheckTx`, is also run: the handler `AnteHandler` does the work specified by each message using a deep copy of the internal state, `checkTxState`, to validate the transaction without modifying the last committed state. The stateful check is able to detect errors such as insufficient funds held by an address or attempted double-spends. Also, `CheckTx` returns `GasUsed` which may or may not be less than the user's provided `GasWanted`.

The transactions approved by a node are held in its [**Mempool**](https://github.com/tendermint/tendermint/blob/75ffa2bf1c7d5805460d941a75112e6a0a38c039/mempool/mempool.go) (memory pool unique to each node) pending approval from the rest of the network.

## Consensus
At each round, a proposer is chosen amongst the validators to create and propose the next block. This validator (presumably honest) has generated a Mempool of validated transactions and now includes them in a block. The validators execute [Tendermint BFT Consensus](https://tendermint.com/docs/spec/consensus/consensus.html#terms); with 2/3 approval from the validators, the block is committed.

## State Changes
During consensus, the validators came to agreement on not only which transactions but also the precise order of the transactions. However, apart from committing to this block in consensus, the ultimate goal is actually for nodes to commit to a new state generated by the transaction state changes. Note that it is also possible for consensus to result in a nil block with no transactions - here, it is assumed that the transaction has made it this far. 
The following ABCI function calls are made in order. While nodes each run everything individually, since the messages' state transitions are deterministic and the order was finalized during consensus, this process yields a single, unambiguous result.
```
		-----------------------		
		| BeginBlock	      |        
		-----------------------		
		          |		
			  v			
		-----------------------		    
		| DeliverTx(tx0)      |  
		| DeliverTx(tx1)      |   	  
		| DeliverTx(tx2)      |  
		| DeliverTx(tx3)      |  
		|	.	      |  
		|	.	      |
		|	.	      |
		-----------------------		
		          |			
			  v			
		-----------------------
		| EndBlock	      |         
		-----------------------
		          |			
			  v			
		-----------------------
		| Commit	      |         
		-----------------------
```
#### BeginBlock
`BeginBlock` is run first, and mainly transmits important data such as block header and Byzantine Validators from the last round of consensus to be used during the next few steps. No transactions are handled here.

#### DeliverTx
The `DeliverTx` function does the bulk of the state change work: it is run for each transaction in the block in sequential order as committed to during consensus. Under the hood, `DeliverTx` is almost identical to `CheckTx` but calls the [`runTx`](https://github.com/cosmos/cosmos-sdk/blob/cec3065a365f03b86bc629ccb6b275ff5846fdeb/baseapp/baseapp.go#L757-L873) function in deliver mode instead of check mode: it utilizes both `AnteHandler` and `MsgHandler`, persisting changes on both `checkTxState` and `deliverTxState`, respectively. If a transaction was not properly validated but somehow made it into the block (i.e. due to a malicious proposer), it is caught here. `BlockGasMeter` is used to keep track of how much gas is left for each transaction; GasUsed is deducted from it and returned in the Response. Any failed state changes resulting from invalid transactions or `BlockGasMeter` running out causes the transaction processing to terminate and any state changes to revert. Any leftover gas is returned to the user.

#### EndBlock
[`EndBlock`](https://github.com/cosmos/cosmos-sdk/blob/9036430f15c057db0430db6ec7c9072df9e92eb2/baseapp/baseapp.go#L875-L886) is always run at the end and is useful for automatic function calls or changing governance/validator parameters. No transactions are handled here.

#### Commit
The application's `Commit` method is run in order to finalize the state changes made by executing this block's transactions. A new state root should be sent back to serve as a merkle proof for the state change. Any application can inherit Baseapp's [`Commit`](https://github.com/cosmos/cosmos-sdk/blob/cec3065a365f03b86bc629ccb6b275ff5846fdeb/baseapp/baseapp.go#L888-L912) method; it synchronizes all the states by writing the `deliverTxState` into the application's internal state, updating both `checkTxState` and `deliverTxState` afterward.

The transaction data itself is still stored in the blockchain, and now its corresponding state changes have been executed and committed. 