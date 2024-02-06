include .env
include go-common.Makefile

# PORTFOLIO_DIR  := <set in .env>
# PORTFOLIO_HTML := <set in .env>  #$(shell /bin/ls $(PORTFOLIO_DIR)/*moneyforward-portfolio.html | tail -n1)
PROMFILE_DIR  := /var/lib/textfile-collector
PROMFILE_PATH := $(PROMFILE_DIR)/mf.prom
PROMFILE_TEMP := /tmp/mf.prom

run: $(BIN)
	PORTFOLIO_HTML=$(PORTFOLIO_HTML) ./$(BIN)

# run-docker:
# 	docker run --rm -it -v $(CURDIR):/app --entrypoint /app/$(BIN) chromedp/headless-shell

update-metrics-file:
	@mkdir -p $(PROMFILE_DIR)
	@test -w $(PROMFILE_PATH)
	PORTFOLIO_HTML=$(PORTFOLIO_HTML) ./$(BIN) > $(PROMFILE_TEMP) && mv $(PROMFILE_TEMP) $(PROMFILE_PATH)
