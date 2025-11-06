.PHONY: bootstrap bootstrap-go bootstrap-node bootstrap-python dev fmt lint test clean

bootstrap: bootstrap-go bootstrap-node bootstrap-python

bootstrap-go:
	@echo "==> Installing Go dependencies"
	@for dir in services/* ; do \
	  if [ -f $$dir/go.mod ]; then \
	    (cd $$dir && go mod tidy); \
	  fi; \
	done

bootstrap-node:
	@echo "==> Installing Node dependencies"
	@if [ -f gateway/package.json ]; then \
	  (cd gateway && npm install); \
	fi

bootstrap-python:
	@echo "==> Setting up Python environments"
	@for dir in ml-services/* ; do \
	  if [ -f $$dir/requirements.txt ]; then \
	    python -m venv $$dir/.venv && $$dir/.venv/Scripts/pip install -r $$dir/requirements.txt; \
	  fi; \
	done

dev:
	@docker compose up --build

fmt:
	@echo "==> Formatting Go code"
	@for dir in services/* ; do \
	  if [ -d $$dir ]; then \
	    gofmt -w $$dir; \
	  fi; \
	done
	@echo "==> Formatting Python code"
	@for dir in ml-services/* ; do \
	  if [ -f $$dir/pyproject.toml ]; then \
	    (cd $$dir && poetry run black .); \
	  fi; \
	done

test:
	@echo "==> Running Go unit tests"
	@for dir in services/* ; do \
	  if [ -f $$dir/go.mod ]; then \
	    (cd $$dir && go test ./...); \
	  fi; \
	done

clean:
	@rm -rf ml-services/*/.venv
