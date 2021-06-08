This is a setup for MONICA. 
It was made to run on ZALF HPC cluster.
It will generate setups, simulate with MONICA and generate images for certain data.

The program pipeline:

run-work-producer.py - generates environments and sends it to MONICA 
run-work-consumer.py - receives the results from MONICA and saves it into csv format
data_to_ascii.go - transforms result data into ascii grids and meta files
create_image_from_ascii.py - make images, png and pdf out of ascii grids.

To run the simulations, use Rundeck
-> HPC Cluster / Start Monica Project

After a run has finish:
- checkout git repository cropMapsEU
- adjust sbatch_go_data_to_ascii.sh with MONICA output path.
- start on cluster: 
  sbatch sbatch_go_data_to_ascii.sh


The setup:
The sub folder json_templates contains MONICA setups.
These setups are put together and changed by run-work-producer.py to provide
different environments to run simulations.
It uses 2 crop rotations - winter wheat / silage maize
and silage maize / winter wheat.
Each runs over 30 years.
Climate scenarios and CO2 values are set in run-work-producer.py code.

Other programs:

asc_mask_data_to_grid.py - used to bring maize/wheat masks into the grid format to be used by data_to_ascii.go.
management_data_to_grid.py - used to generate auto sowing and harvest dates for the crop rotation inserted by run-work-producer.py