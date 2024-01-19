SHELL := /bin/bash

TARGETS := part11 part12 part21 part22

CURDIR := $(shell pwd)
GO_DIR := $(CURDIR)/go-workspace
PY_DIR := $(CURDIR)/py-scripts
TEX_DIR := $(CURDIR)/report

IMAGE_NET_TEST_DIR := /tiny-imagenet-200/test/images

TEMP_DIR := $(CURDIR)/temp
DATA_DIR := $(TEMP_DIR)/data
FIG_DIR := $(TEMP_DIR)/fig
IMG_DIR := /osdata/osgroup4/generated_imgs

DIRS := $(TEMP_DIR) $(DATA_DIR) $(FIG_DIR) $(IMG_DIR)

TEX_FLAGS := -synctex=1 -interaction=nonstopmode -file-line-error -output-directory=$(TEMP_DIR)

empty :=
space := $(empty) $(empty)
comma := ,
dollar := $$

all: $(TARGETS)

part11: $(CURDIR)/part11.pdf

part12: $(CURDIR)/part12.pdf

part21:
	make deploy_server
	make start_server
	make part21-experiment
	make part21-plot
	make stop_server
	make $(CURDIR)/part21.pdf

# You can also write
# part21: deploy_server start_server part21-experiment part21-plot stop_server $(CURDIR)/part21.pdf
# In this way, you cannot use make -j to run the targets in parallel

part22:
	make deploy_server
	make start_server
	make part22-experiment
	make part22-plot
	make stop_server
	make $(CURDIR)/part22.pdf

SophiaCoin: $(TEMP_DIR)/daemon $(TEMP_DIR)/client $(TEMP_DIR)/parser $(CURDIR)/project2.pdf

clean:
	rm -rf $(TEMP_DIR)

.PHONY: all clean part11 part12 test

$(DIRS):
	mkdir -p $@

# The following are for project part11

$(TEMP_DIR)/part11: $(wildcard $(GO_DIR)/part11/*.go)
	cd $(GO_DIR)/part11 && go mod tidy && go build -o $(TEMP_DIR)/part11

define PART11_EXPERIMENT_TARGET = 
$(DATA_DIR)/cap_$(1)_threads_$(2).txt
endef

define PART11_EXPERIMENT =
$(call PART11_EXPERIMENT_TARGET,$(1),$(2)): $(TEMP_DIR)/part11 $(DATA_DIR)
	$(TEMP_DIR)/part11 -cap $(1) -n-t $(2) -dir $(DATA_DIR)
endef

# construct the experiments
$(foreach log_cap,$(shell seq 0 9),\
	$(foreach threads,$(shell seq 1 10),\
		$(eval $(call PART11_EXPERIMENT,$(shell echo "2^$(log_cap)" | bc),$(threads)))\
	)\
)

LOG_CAP := 5 # capacity = 2^0, 2^1, ..., 2^LOG_CAP
THREADS := 6 # number of threads = 1, 2, ..., THREADS
THROUGHPUT_FIG_EXPERIMENTS := \
$(foreach log_cap,$(shell seq 0 $(LOG_CAP)),\
	$(foreach threads,$(shell seq 1 $(THREADS)),\
		$(call PART11_EXPERIMENT_TARGET,$(shell echo "2^$(log_cap)" | bc),$(threads))\
	)\
)

$(FIG_DIR)/part11_throughput_fig.png: $(THROUGHPUT_FIG_EXPERIMENTS) $(FIG_DIR)
	python3 $(PY_DIR)/plot-throughput.py --data-dir=$(DATA_DIR) --out-dir=$(FIG_DIR)\
		--log-cap=$(LOG_CAP) --threads=$(THREADS)

define PART11_EXPERIMENT_CDF_PLOT_TARGET =
$(FIG_DIR)/part11_cap_$(1)_threads_$(2)_cdf.png
endef

define PART11_EXPERIMENT_CDF_PLOT =
$(call PART11_EXPERIMENT_CDF_PLOT_TARGET,$(1),$(2)): $(call PART11_EXPERIMENT_TARGET,$(1),$(2)) $(FIG_DIR)
	python3 $(PY_DIR)/plot-cdf.py --data-file $(call PART11_EXPERIMENT_TARGET,$(1),$(2))\
		--out-file $(call PART11_EXPERIMENT_CDF_PLOT_TARGET,$(1),$(2)) --cap $(1) --threads $(2)
endef

$(foreach log_cap,$(shell seq 0 $(LOG_CAP)),\
	$(foreach threads,$(shell seq 1 $(THREADS)),\
		$(eval $(call PART11_EXPERIMENT_CDF_PLOT,$(shell echo "2^$(log_cap)" | bc),$(threads)))\
	)\
)

CDF_FIG_TARGETS := \
$(foreach log_cap,$(shell seq 0 $(LOG_CAP)),\
	$(foreach threads,$(shell seq 1 $(THREADS)),\
		$(call PART11_EXPERIMENT_CDF_PLOT_TARGET,$(shell echo "2^$(log_cap)" | bc),$(threads))\
	)\
)

$(CURDIR)/part11.pdf: $(TEX_DIR)/part11.tex $(CDF_FIG_TARGETS) $(FIG_DIR)/part11_throughput_fig.png
	cp $< $(TEMP_DIR)
	sed -i 's/\\newcommand{\\FIGDIR}{}/\\newcommand{\\FIGDIR}{$(subst /,\/,$(FIG_DIR))}/g' $(TEMP_DIR)/part11.tex
	pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part11.tex
	pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part11.tex
	mv $(TEMP_DIR)/part11.pdf $@


# The following are for part12

$(TEMP_DIR)/part12: $(wildcard $(GO_DIR)/part12/*.go)
	cd $(GO_DIR)/part12 && go mod tidy && go build -o $(TEMP_DIR)/part12

$(DATA_DIR)/part12.csv: $(TEMP_DIR)/part12 $(DATA_DIR) $(IMG_DIR)
	@echo "\e[31mCAUTION: The imagenet test split directory is in $(IMAGE_NET_TEST_DIR)\e[0m"
	@echo "\e[31mCAUTION: The output directory is in $(IMG_DIR)\e[0m"
	@touch $(DATA_DIR)/part12.csv
	@echo "cpus,capacity,threads,time(s)" > $(DATA_DIR)/part12.csv

# baseline
	@echo "Running baseline..."
	@cap=1; cpu=1; threads=1; \
	printf "$$cpu,$$cap,$$threads," >> $(DATA_DIR)/part12.csv;\
	$(TEMP_DIR)/part12 -cpus $$cpu -cap $$cap -n-t $$threads\
		-out-dir $(IMG_DIR) -src-dir $(IMAGE_NET_TEST_DIR)\
		>> $(DATA_DIR)/part12.csv;
	
# varying number of threads
	@echo "Running experiments on varying number of threads..."
	@cap=100; cpu=24; \
	for threads in 1 2 4 6 8 10 16 24; do \
		rm -rf $(IMG_DIR)/*; \
		printf "$$cpu,$$cap,$$threads," >> $(DATA_DIR)/part12.csv;\
		$(TEMP_DIR)/part12 -cpus $$cpu -cap $$cap -n-t $$threads\
			-out-dir $(IMG_DIR) -src-dir $(IMAGE_NET_TEST_DIR)\
			>> $(DATA_DIR)/part12.csv; \
	done; \

# varying number of cpus
	@echo "Running experiments on varying number of cpus..."
	@cap=100; threads=8; \
	for cpu in 1 2 4 6 8 10 16 24; do \
		rm -rf $(IMG_DIR)/*; \
		printf "$$cpu,$$cap,$$threads," >> $(DATA_DIR)/part12.csv;\
		$(TEMP_DIR)/part12 -cpus $$cpu -cap $$cap -n-t $$threads\
			-out-dir $(IMG_DIR) -src-dir $(IMAGE_NET_TEST_DIR)\
			>> $(DATA_DIR)/part12.csv; \
	done; \

# varying capacity
	@echo "Running experiments on varying capacity of queue..."
	@cpu=24; threads=24; \
	for cap in 1 10 100 1000; do \
		rm -rf $(IMG_DIR)/*; \
		printf "$$cpu,$$cap,$$threads," >> $(DATA_DIR)/part12.csv;\
		$(TEMP_DIR)/part12 -cpus $$cpu -cap $$cap -n-t $$threads\
			-out-dir $(IMG_DIR) -src-dir $(IMAGE_NET_TEST_DIR)\
			>> $(DATA_DIR)/part12.csv; \
	done;

PART12_FIGS := $(FIG_DIR)/part12_threads.png $(FIG_DIR)/part12_cpus.png $(FIG_DIR)/part12_cap.png

$(PART12_FIGS): $(DATA_DIR)/part12.csv $(FIG_DIR)
	python3 $(PY_DIR)/part12-plot.py --data-file $< --out-dir $(FIG_DIR)
	
$(CURDIR)/part12.pdf: $(TEX_DIR)/part12.tex $(PART12_FIGS)
	cp $< $(TEMP_DIR)
	sed -i 's/\\newcommand{\\FIGDIR}{}/\\newcommand{\\FIGDIR}{$(subst /,\/,$(FIG_DIR))}/g' $(TEMP_DIR)/part12.tex
	pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part12.tex
	pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part12.tex
	mv $(TEMP_DIR)/part12.pdf $@


# The following are for part2

$(TEMP_DIR)/server: $(wildcard $(GO_DIR)/part2/cmd/*/*.go) $(GO_DIR)/part2/imgdownload/imgdownload.proto
	@echo "Building server..."
	@cd $(GO_DIR) && \
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./part2/imgdownload/imgdownload.proto
	@cd $(GO_DIR)/part2/cmd/server && go mod tidy && go build -o $(TEMP_DIR)/server
	@echo "Done building server"

$(TEMP_DIR)/client21: $(wildcard $(GO_DIR)/part2/cmd/client21/*.go) $(GO_DIR)/part2/imgdownload/imgdownload.proto
	@echo "Building client21..."
	@cd $(GO_DIR) && \
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./part2/imgdownload/imgdownload.proto
	@cd $(GO_DIR)/part2/cmd/client21 && go mod tidy && go build -o $(TEMP_DIR)/client21
	@echo "Done building client21"

$(TEMP_DIR)/client22: $(wildcard $(GO_DIR)/part2/cmd/client22/*.go) $(GO_DIR)/part2/imgdownload/imgdownload.proto
	@echo "Building client22..."
	@cd $(GO_DIR) && \
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative \
		./part2/imgdownload/imgdownload.proto
	@cd $(GO_DIR)/part2/cmd/client22 && go mod tidy && go build -o $(TEMP_DIR)/client22
	@echo "Done building client22"

deploy_server: $(TEMP_DIR)/server
	@echo "Deploying server..."
	@echo "Done"

start_server: deploy_server
	@echo "Starting server..."
	@ports=(`seq 8051 8070`); \
	hostnums=(`for i in {0..19}; do ssh osgroup4@122.200.68.26 -p $${ports[$$i]} "hostname | sed 's/[^0-9]//g'"; done`); \
	ipaddrs=(`for h in $${hostnums[@]}; do echo "10.1.0.$$h:51151"; done`); \
	threads=(1 2 4 8 12 24 24 24 24 24 24 24 24 24 24 24 24 24 24 24); \
	for i in {0..18}; do \
		pid=$$(ssh osgroup4@122.200.68.26 -p $${ports[$$i]} "$(TEMP_DIR)/server -n-t $${threads[$$i]} -addr $${ipaddrs[$$i]} > /dev/null 2>&1 & echo \$(dollar)!"); \
		echo "Started server(pid $${pid} on machine at port $${ports[$$i]}) on ip $${ipaddrs[$$i]}, with $${threads[$$i]} threads running..."; \
	done; 
	@echo "Done"

stop_server:
	@echo "Stopping server..."
	@ports=(`seq 8051 8070`); \
	for i in {0..18}; do \
		pid=$$(ssh osgroup4@122.200.68.26 -p $${ports[$$i]} "lsof -i:51151 | grep server | grep LISTEN | awk '{print \$$2}'"); \
		ssh osgroup4@122.200.68.26 -p $${ports[$$i]} "kill $$pid"; \
		echo "Stopped server(pid $$pid on machine at port $${ports[$$i]})..."; \
	done;
	@echo "Done"

part21-experiment: $(TEMP_DIR)/client21 $(PY_DIR)/part21-experiment.py
	@echo "Running part21 experiments..."
	@echo "\e[31mCAUTION: The output directory of clients' downloaded images is in $(IMG_DIR)\e[0m"
	@mkdir -p $(IMG_DIR) $(FIG_DIR) $(DATA_DIR)
	@echo "Recording log to $(TEMP_DIR)/part21-experiment.log"
	@python3 $(PY_DIR)/part21-experiment.py --out-dir $(IMG_DIR) > $(TEMP_DIR)/part21-experiment.log 2>&1
	@echo "Done"

part21-plot: $(PY_DIR)/part21-plot.py
	@echo "Plotting part21 results..."
	@python3 $(PY_DIR)/part21-plot.py
	@echo "Done"

part22-experiment: $(TEMP_DIR)/client22 $(PY_DIR)/part22-experiment.py
	@echo "Running part22 experiments..."
	@echo "\e[31mCAUTION: The output directory of clients' downloaded images is in $(IMG_DIR)\e[0m"
	@mkdir -p $(IMG_DIR) $(FIG_DIR) $(DATA_DIR)
	@echo "Recording log to $(TEMP_DIR)/part22-experiment.log"
	@python3 $(PY_DIR)/part22-experiment.py --out-dir $(IMG_DIR) > $(TEMP_DIR)/part22-experiment.log 2>&1
	@echo "Done"

part22-plot: $(PY_DIR)/part22-plot.py
	@echo "Plotting part22 results..."
	@python3 $(PY_DIR)/part22-plot.py
	@echo "Done"

$(CURDIR)/part21.pdf: $(TEX_DIR)/part21.tex
	cp $< $(TEMP_DIR)
	cp $(TEX_DIR)/fig/* $(FIG_DIR)
	sed -i 's/\\newcommand{\\FIGDIR}{}/\\newcommand{\\FIGDIR}{$(subst /,\/,$(FIG_DIR))}/g' $(TEMP_DIR)/part21.tex
	-pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part21.tex > /dev/null 2>&1
	-pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part21.tex > /dev/null 2>&1
	mv $(TEMP_DIR)/part21.pdf $@

$(CURDIR)/part22.pdf: $(TEX_DIR)/part22.tex
	cp $< $(TEMP_DIR)
	cp $(TEX_DIR)/fig/* $(FIG_DIR)
	sed -i 's/\\newcommand{\\FIGDIR}{.*}/\\newcommand{\\FIGDIR}{$(subst /,\/,$(FIG_DIR))}/g' $(TEMP_DIR)/part22.tex
	-pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part22.tex > /dev/null 2>&1
	-pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part22.tex > /dev/null 2>&1
	mv $(TEMP_DIR)/part22.pdf $@


$(GO_DIR)/SophiaCoin/pkg/rpc/broadcast.pb.go: $(GO_DIR)/SophiaCoin/pkg/rpc/broadcast.proto
	cd $(GO_DIR)/SophiaCoin/pkg/rpc && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative broadcast.proto

$(GO_DIR)/SophiaCoin/pkg/rpc/broadcast_grpc.pb.go: $(GO_DIR)/SophiaCoin/pkg/rpc/broadcast.proto
	cd $(GO_DIR)/SophiaCoin/pkg/rpc && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative broadcast.proto

SophiaCoinDependency := $(wildcard $(GO_DIR)/SophiaCoin/pkg/*/*.go) $(GO_DIR)/SophiaCoin/pkg/rpc/broadcast.pb.go $(GO_DIR)/SophiaCoin/pkg/rpc/broadcast_grpc.pb.go

$(TEMP_DIR)/daemon: $(GO_DIR)/SophiaCoin/cmd/daemon/*.go $(SophiaCoinDependency)
	cd $(GO_DIR)/SophiaCoin/cmd/daemon && go mod tidy && go build -o $(TEMP_DIR)/daemon

$(TEMP_DIR)/client: $(GO_DIR)/SophiaCoin/cmd/client/*.go $(SophiaCoinDependency)
	cd $(GO_DIR)/SophiaCoin/cmd/client && go mod tidy && go build -o $(TEMP_DIR)/client

$(TEMP_DIR)/parser: $(GO_DIR)/SophiaCoin/cmd/parser/*.go $(SophiaCoinDependency)
	cd $(GO_DIR)/SophiaCoin/cmd/parser && go mod tidy && go build -o $(TEMP_DIR)/parser

$(CURDIR)/project2.pdf: $(TEX_DIR)/project2.tex $(TEX_DIR)/ref.bib
	cp $^ $(TEMP_DIR)
	cp $(TEX_DIR)/fig/* $(FIG_DIR)
	sed -i 's/\\newcommand{\\FIGDIR}{.*}/\\newcommand{\\FIGDIR}{$(subst /,\/,$(FIG_DIR))}/g' $(TEMP_DIR)/project2.tex
	-pdflatex $(TEX_FLAGS) $(TEMP_DIR)/project2.tex > /dev/null 2>&1
	-cd $(TEMP_DIR); bibtex project2 > /dev/null 2>&1
	-pdflatex $(TEX_FLAGS) $(TEMP_DIR)/project2.tex > /dev/null 2>&1
	-pdflatex $(TEX_FLAGS) $(TEMP_DIR)/project2.tex > /dev/null 2>&1
	mv $(TEMP_DIR)/project2.pdf $@
