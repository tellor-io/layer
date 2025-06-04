import re
import sys

# Basic template for RegisterServices
REGISTER_SERVICES_TEMPLATE = '''func (am AppModule) RegisterServices(cfg module.Configurator) {{
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQuerier(am.keeper))

	m := keeper.NewMigrator(am.keeper)
	if err := cfg.RegisterMigration(types.ModuleName, {old_version}, m.MigrateFork); err != nil {{
		panic(err)
	}}
'''

def update_consensus_version(file_path):
    # Read the file
    with open(file_path, 'r') as f:
        content = f.read()

    # Find the consensus version
    consensus_pattern = r'func \(AppModule\) ConsensusVersion\(\) uint64 \{ return (\d+) \}'
    consensus_match = re.search(consensus_pattern, content)
    
    if not consensus_match:
        print("Could not find consensus version")
        return False
    
    old_version = int(consensus_match.group(1))
    new_version = old_version + 1
    
    # Update the consensus version
    new_content = re.sub(
        consensus_pattern,
        f'func (AppModule) ConsensusVersion() uint64 {{ return {new_version} }}',
        content
    )
    
    # Find the RegisterServices function using a more precise pattern
    register_services_pattern = r'func \(am AppModule\) RegisterServices\(cfg module\.Configurator\) \{[^}]*\}'
    register_services_match = re.search(register_services_pattern, new_content, re.DOTALL)
    
    if register_services_match:
        # Replace the entire function with our template
        new_register_services = REGISTER_SERVICES_TEMPLATE.format(old_version=old_version)
        new_content = re.sub(register_services_pattern, new_register_services, new_content, flags=re.DOTALL)
        print(f"Updated RegisterServices function with version {old_version}")
    else:
        print("No RegisterServices function found - this is normal for some modules")
    
    # Write the updated content back to the file
    with open(file_path, 'w') as f:
        f.write(new_content)
    
    print(f"Updated consensus version from {old_version} to {new_version}")
    return True

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python update_consensus.py <file_path>")
        sys.exit(1)
    
    file_path = sys.argv[1]
    if not update_consensus_version(file_path):
        sys.exit(1) 