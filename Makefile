TARGETS := part11 part12

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

all: $(TARGETS)

part11: $(CURDIR)/part11.pdf

part12: $(CURDIR)/part12.pdf

clean:
	rm -rf $(TEMP_DIR)

.PHONY: all clean part11 part12 test

$(DIRS):
	mkdir -p $@

# The following are for project part11

$(TEMP_DIR)/part11: $(TEMP_DIR) $(wildcard $(GO_DIR)/part11/*.go)
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
	sed -i 's/\\newcommand{\\DATADIR}{}/\\newcommand{\\DATADIR}{$(subst /,\/,$(DATA_DIR))}/g' $(TEMP_DIR)/part11.tex
	pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part11.tex
	pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part11.tex
	mv $(TEMP_DIR)/part11.pdf $@


# The following are for part12

$(TEMP_DIR)/part12: $(TEMP_DIR) $(wildcard $(GO_DIR)/part12/*.go)
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
	sed -i 's/\\newcommand{\\DATADIR}{}/\\newcommand{\\DATADIR}{$(subst /,\/,$(DATA_DIR))}/g' $(TEMP_DIR)/part12.tex
	pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part12.tex
	pdflatex $(TEX_FLAGS) $(TEMP_DIR)/part12.tex
	mv $(TEMP_DIR)/part12.pdf $@
