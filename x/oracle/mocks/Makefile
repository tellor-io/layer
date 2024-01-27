mock-clean:
	@rm -f ./*.go

mock-gen:
	@go run github.com/vektra/mockery/v2 --name=AccountKeeper --dir=../types --recursive --output=.
	@go run github.com/vektra/mockery/v2 --name=BankKeeper --dir=../types --recursive --output=.
	@go run github.com/vektra/mockery/v2 --name=DistrKeeper --dir=../types --recursive --output=.
	
