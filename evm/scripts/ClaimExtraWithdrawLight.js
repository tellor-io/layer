require("dotenv").config();
const { ethers } = require("ethers");

// Lightweight version using ethers.js directly (no Hardhat dependency)
// Usage: node scripts/ClaimExtraWithdrawLight.js [--dry-run] [--once]
// Or with args: node scripts/ClaimExtraWithdrawLight.js --private-key <key> --recipient <address> --node-url <url> [--dry-run] [--once]
// Environment variables (default): TESTNET_PK, NODE_URL_SEPOLIA_TESTNET, RECIPIENT_ADDRESS

const TOKEN_BRIDGE_ADDRESS = "0xE4A37df3755D034287817970ccdD5222dE13432C";
const CHECK_INTERVAL_HOURS = 11;
const CHECK_INTERVAL_MS = CHECK_INTERVAL_HOURS * 60 * 60 * 1000;
const POLL_INTERVAL_MINUTES = 10;
const POLL_INTERVAL_MS = POLL_INTERVAL_MINUTES * 60 * 1000;

// Minimal ABI - only the functions we need
const TOKEN_BRIDGE_ABI = [
    "function claimExtraWithdraw(address _recipient) external",
    "function tokensToClaim(address recipient) public view returns (uint256)",
    "function bridgeState() public view returns (uint8)",
    "function initialized() public view returns (bool)",
    "function withdrawLimit() external view returns (uint256)",
    "event ExtraWithdrawClaimed(address indexed _recipient, uint256 _amount)"
];

// Parse command line arguments
const args = process.argv.slice(2);
const argMap = {};
const flags = new Set();

// Parse arguments: flags (--flag) and key-value pairs (--key value)
for (let i = 0; i < args.length; i++) {
    if (args[i].startsWith('--')) {
        const key = args[i].replace(/^--/, '');
        const nextArg = args[i + 1];
        
        // If next argument doesn't exist or is also a flag, this is a flag
        if (!nextArg || nextArg.startsWith('--')) {
            flags.add(key);
        } else {
            // Otherwise it's a key-value pair
            argMap[key] = nextArg;
            i++; // Skip the value in next iteration
        }
    }
}

// Check for flags
const isDryRun = flags.has('dry-run');
const runOnce = flags.has('once');

// Get configuration from command line args, then environment variables
// Priority: TESTNET_PK/NODE_URL_SEPOLIA_TESTNET (default), then fallback to generic names
const PRIVATE_KEY = argMap['private-key'] || process.env.TESTNET_PK || process.env.MAINNET_PK || process.env.PRIVATE_KEY;
const NODE_URL = argMap['node-url'] || process.env.NODE_URL_SEPOLIA_TESTNET || process.env.NODE_URL_MAINNET || process.env.NODE_URL;
const RECIPIENT_ADDRESS = argMap['recipient'] || process.env.RECIPIENT_ADDRESS;

// Validate required configuration
if (!PRIVATE_KEY) {
    console.error("Error: Private key required.");
    console.error("Usage: node scripts/ClaimExtraWithdrawLight.js [--private-key <key>] [--recipient <address>] [--node-url <url>] [--dry-run] [--once]");
    console.error("Or set environment variables: TESTNET_PK (or MAINNET_PK/PRIVATE_KEY), RECIPIENT_ADDRESS, NODE_URL_SEPOLIA_TESTNET (or NODE_URL_MAINNET/NODE_URL)");
    process.exit(1);
}

if (!NODE_URL) {
    console.error("Error: Node URL required.");
    console.error("Usage: node scripts/ClaimExtraWithdrawLight.js [--private-key <key>] [--recipient <address>] [--node-url <url>] [--dry-run] [--once]");
    console.error("Or set environment variables: TESTNET_PK (or MAINNET_PK/PRIVATE_KEY), RECIPIENT_ADDRESS, NODE_URL_SEPOLIA_TESTNET (or NODE_URL_MAINNET/NODE_URL)");
    process.exit(1);
}

if (!RECIPIENT_ADDRESS) {
    console.error("Error: Recipient address required.");
    console.error("Usage: node scripts/ClaimExtraWithdrawLight.js [--private-key <key>] [--recipient <address>] [--node-url <url>] [--dry-run] [--once]");
    console.error("This should be the address that will receive the claimed tokens (usually your wallet address).");
    process.exit(1);
}

// Validate that recipient is not the contract address
if (RECIPIENT_ADDRESS.toLowerCase() === TOKEN_BRIDGE_ADDRESS.toLowerCase()) {
    console.error("⚠️  Error: Recipient address cannot be the TokenBridge contract address!");
    console.error(`   TokenBridge address: ${TOKEN_BRIDGE_ADDRESS}`);
    console.error(`   Recipient address:   ${RECIPIENT_ADDRESS}`);
    console.error("   The recipient should be YOUR wallet address that will receive the tokens.");
    console.error("   This is typically the same as your wallet address (the one signing the transaction).");
    process.exit(1);
}

async function verifyConnection() {
    try {
        console.log(`\n[${new Date().toISOString()}] Verifying connection...`);
        
        // Connect to the network
        const provider = new ethers.providers.JsonRpcProvider(NODE_URL);
        const wallet = new ethers.Wallet(PRIVATE_KEY, provider);
        
        console.log(`✅ Connected to network: ${NODE_URL}`);
        console.log(`✅ Wallet address (signer): ${wallet.address}`);
        console.log(`✅ Recipient address (will receive tokens): ${RECIPIENT_ADDRESS}`);
        console.log(`✅ Network chain ID: ${(await provider.getNetwork()).chainId}`);
        
        // Get wallet balance
        const balance = await provider.getBalance(wallet.address);
        console.log(`✅ Wallet ETH balance: ${ethers.utils.formatEther(balance)} ETH`);
        
        // Warn if recipient is different from wallet address (might be intentional, but worth noting)
        if (RECIPIENT_ADDRESS.toLowerCase() !== wallet.address.toLowerCase()) {
            console.log(`ℹ️  Note: Recipient address differs from wallet address. Tokens will be sent to recipient.`);
        }
        
        // Connect to contract
        const tokenBridge = new ethers.Contract(TOKEN_BRIDGE_ADDRESS, TOKEN_BRIDGE_ABI, wallet);
        
        // Verify contract exists
        const code = await provider.getCode(TOKEN_BRIDGE_ADDRESS);
        if (code === "0x") {
            throw new Error("No contract code found at address. Contract may not be deployed.");
        }
        console.log(`✅ Contract found at: ${TOKEN_BRIDGE_ADDRESS}`);
        
        // Check contract state
        const initialized = await tokenBridge.initialized();
        console.log(`✅ Bridge initialized: ${initialized}`);
        
        const bridgeState = await tokenBridge.bridgeState();
        const stateNames = ["NORMAL", "PAUSED", "UNPAUSED"];
        console.log(`✅ Bridge state: ${stateNames[bridgeState]} (${bridgeState})`);
        
        if (bridgeState === 1) {
            console.log("⚠️  Warning: Bridge is PAUSED. Cannot claim withdraws.");
        }
        
        // Check withdraw limit
        const withdrawLimit = await tokenBridge.withdrawLimit();
        console.log(`✅ Withdraw limit: ${ethers.utils.formatEther(withdrawLimit)} TRB`);
        
        // Check tokens available to claim
        const tokensToClaim = await tokenBridge.tokensToClaim(RECIPIENT_ADDRESS);
        console.log(`✅ Tokens available to claim: ${ethers.utils.formatEther(tokensToClaim)} TRB`);
        
        if (tokensToClaim.eq(0)) {
            console.log("⚠️  No tokens available to claim at this time.");
        }
        
        if (withdrawLimit.eq(0)) {
            console.log("⚠️  Withdraw limit is 0. Will need to wait for limit to refresh.");
        }
        
        // Estimate gas (dry run)
        if (!isDryRun && tokensToClaim.gt(0) && bridgeState !== 1) {
            try {
                const gasEstimate = await tokenBridge.estimateGas.claimExtraWithdraw(RECIPIENT_ADDRESS);
                console.log(`✅ Gas estimate: ${gasEstimate.toString()}`);
                
                const gasPrice = await provider.getGasPrice();
                const estimatedCost = gasEstimate.mul(gasPrice);
                console.log(`✅ Estimated transaction cost: ${ethers.utils.formatEther(estimatedCost)} ETH`);
                console.log(`   (Gas price: ${ethers.utils.formatUnits(gasPrice, "gwei")} gwei)`);
                
                if (balance.lt(estimatedCost)) {
                    console.log("⚠️  Warning: Wallet balance may be insufficient for gas fees.");
                }
            } catch (error) {
                console.log(`⚠️  Could not estimate gas: ${error.message}`);
            }
        }
        
        console.log("\n✅ Connection verification complete!");
        return true;
        
    } catch (error) {
        console.error(`\n❌ Connection verification failed:`, error.message);
        
        if (error.message.includes("invalid response")) {
            console.error("   Check that your NODE_URL is correct and accessible.");
        } else if (error.message.includes("invalid private key")) {
            console.error("   Check that your PRIVATE_KEY is correct.");
        } else if (error.message.includes("insufficient funds")) {
            console.error("   Your wallet may not have enough ETH for gas.");
        }
        
        return false;
    }
}

async function claimExtraWithdraw() {
    try {
        // Connect to the network
        const provider = new ethers.providers.JsonRpcProvider(NODE_URL);
        const wallet = new ethers.Wallet(PRIVATE_KEY, provider);
        
        console.log(`\n[${new Date().toISOString()}] Processing claim...`);
        console.log(`Wallet address: ${wallet.address}`);
        console.log(`Recipient address: ${RECIPIENT_ADDRESS}`);
        console.log(`TokenBridge address: ${TOKEN_BRIDGE_ADDRESS}`);
        
        // Connect to contract
        const tokenBridge = new ethers.Contract(TOKEN_BRIDGE_ADDRESS, TOKEN_BRIDGE_ABI, wallet);
        
        // Check if there are tokens to claim
        const tokensToClaim = await tokenBridge.tokensToClaim(RECIPIENT_ADDRESS);
        console.log(`Tokens available to claim: ${ethers.utils.formatEther(tokensToClaim)} TRB`);
        
        if (tokensToClaim.eq(0)) {
            console.log("No tokens available to claim. Skipping transaction.");
            return false;
        }
        
        // Check bridge state
        const bridgeState = await tokenBridge.bridgeState();
        if (bridgeState === 1) { // BridgeState.PAUSED
            console.log("Warning: Bridge is PAUSED. Cannot claim withdraws.");
            return false;
        }
        
        // Estimate gas
        const gasEstimate = await tokenBridge.estimateGas.claimExtraWithdraw(RECIPIENT_ADDRESS);
        console.log(`Estimated gas: ${gasEstimate.toString()}`);
        
        // Get current gas price
        const gasPrice = await provider.getGasPrice();
        console.log(`Current gas price: ${ethers.utils.formatUnits(gasPrice, "gwei")} gwei`);
        
        // Send transaction
        console.log("Sending claimExtraWithdraw transaction...");
        const tx = await tokenBridge.claimExtraWithdraw(RECIPIENT_ADDRESS, {
            gasLimit: gasEstimate.mul(120).div(100), // Add 20% buffer
        });
        
        console.log(`Transaction hash: ${tx.hash}`);
        console.log("Waiting for confirmation...");
        
        // Wait for transaction to be mined
        const receipt = await tx.wait();
        
        console.log(`\n✅ Transaction confirmed!`);
        console.log(`Block number: ${receipt.blockNumber}`);
        console.log(`Gas used: ${receipt.gasUsed.toString()}`);
        
        // Parse events
        const event = receipt.events?.find(e => e.event === "ExtraWithdrawClaimed");
        if (event) {
            console.log(`Amount claimed: ${ethers.utils.formatEther(event.args._amount)} TRB`);
        }
        
        // Check remaining tokens to claim
        const remainingTokens = await tokenBridge.tokensToClaim(RECIPIENT_ADDRESS);
        console.log(`Remaining tokens to claim: ${ethers.utils.formatEther(remainingTokens)} TRB`);
        
        return true;
        
    } catch (error) {
        console.error(`\n❌ Error claiming extra withdraw:`, error.message);
        
        // Provide helpful error messages
        if (error.message.includes("bridge is paused")) {
            console.error("The bridge is currently paused. Please wait for it to be unpaused.");
        } else if (error.message.includes("amount must be > 0")) {
            console.error("No tokens available to claim at this time.");
        } else if (error.message.includes("withdraw limit must be > 0")) {
            console.error("Withdraw limit is currently 0. Please wait for the limit to refresh.");
        } else if (error.message.includes("insufficient funds")) {
            console.error("Insufficient funds for gas. Please add ETH to your wallet.");
        } else if (error.message.includes("nonce")) {
            console.error("Nonce error. This may resolve on the next attempt.");
        }
        
        return false;
    }
}

// Function to check withdraw limit
async function checkWithdrawLimit() {
    try {
        const provider = new ethers.providers.JsonRpcProvider(NODE_URL);
        const wallet = new ethers.Wallet(PRIVATE_KEY, provider);
        const tokenBridge = new ethers.Contract(TOKEN_BRIDGE_ADDRESS, TOKEN_BRIDGE_ABI, wallet);
        
        const withdrawLimit = await tokenBridge.withdrawLimit();
        return withdrawLimit;
    } catch (error) {
        console.error(`Error checking withdraw limit: ${error.message}`);
        return null;
    }
}

// Function to poll withdraw limit until it's > 0
async function waitForWithdrawLimit() {
    console.log(`\n[${new Date().toISOString()}] Checking withdraw limit...`);
    
    while (true) {
        const withdrawLimit = await checkWithdrawLimit();
        
        if (withdrawLimit === null) {
            console.log("⚠️  Error checking withdraw limit. Retrying in 10 minutes...");
            await new Promise(resolve => setTimeout(resolve, POLL_INTERVAL_MS));
            continue;
        }
        
        const limitValue = ethers.utils.formatEther(withdrawLimit);
        console.log(`   Withdraw limit: ${limitValue} TRB`);
        
        if (withdrawLimit.gt(0)) {
            console.log(`✅ Withdraw limit is now > 0. Proceeding with claim...`);
            return true;
        }
        
        console.log(`   Withdraw limit is 0. Waiting ${POLL_INTERVAL_MINUTES} minutes before checking again...`);
        await new Promise(resolve => setTimeout(resolve, POLL_INTERVAL_MS));
    }
}

// Function to run the claim and schedule the next one
async function runAndSchedule() {
    // Always verify connection first
    const verified = await verifyConnection();
    
    if (!verified) {
        console.error("\n❌ Connection verification failed. Exiting.");
        process.exit(1);
    }
    
    // If dry run, exit after verification
    if (isDryRun) {
        console.log("\n✅ Dry run complete. No transactions were sent.");
        process.exit(0);
    }
    
    // Check withdraw limit first
    console.log(`\n[${new Date().toISOString()}] Checking withdraw limit (every ${CHECK_INTERVAL_HOURS} hours)...`);
    const withdrawLimit = await checkWithdrawLimit();
    
    if (withdrawLimit === null) {
        console.error("❌ Failed to check withdraw limit. Exiting.");
        process.exit(1);
    }
    
    const limitValue = ethers.utils.formatEther(withdrawLimit);
    console.log(`Current withdraw limit: ${limitValue} TRB`);
    
    // If withdraw limit is 0, poll every 10 minutes until it's > 0
    if (withdrawLimit.eq(0)) {
        console.log(`Withdraw limit is 0. Polling every ${POLL_INTERVAL_MINUTES} minutes until limit > 0...`);
        await waitForWithdrawLimit();
    }
    
    // Now proceed with actual claim
    await claimExtraWithdraw();
    
    if (runOnce) {
        console.log("\n✅ One-time execution complete. Exiting.");
        process.exit(0);
    }
    
    // Schedule next execution in 11 hours
    const nextRun = new Date(Date.now() + CHECK_INTERVAL_MS);
    console.log(`\n⏰ Next withdraw limit check scheduled for: ${nextRun.toISOString()}`);
    console.log(`   (in ${CHECK_INTERVAL_HOURS} hours)\n`);
    
    setTimeout(runAndSchedule, CHECK_INTERVAL_MS);
}

// Main execution
console.log("=".repeat(60));
console.log("TokenBridge Extra Withdraw Claimer (Lightweight)");
console.log("=".repeat(60));
console.log(`Mode: ${isDryRun ? 'DRY RUN (verification only)' : runOnce ? 'One-time execution' : `Checking withdraw limit every ${CHECK_INTERVAL_HOURS} hours`}`);
console.log(`TokenBridge: ${TOKEN_BRIDGE_ADDRESS}`);
console.log(`Recipient: ${RECIPIENT_ADDRESS}`);
console.log("=".repeat(60));

// Run immediately, then schedule recurring executions (unless flags)
runAndSchedule().catch(error => {
    console.error("Fatal error:", error);
    process.exit(1);
});
