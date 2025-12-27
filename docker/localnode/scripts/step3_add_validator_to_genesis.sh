#!/bin/bash

jq '.validators = []' ~/.tabi/config/genesis.json > ~/.tabi/config/tmp_genesis.json
cd build/generated/gentx
IDX=0
for FILE in *
do
    jq '.validators['$IDX'] |= .+ {}' ~/.tabi/config/tmp_genesis.json > ~/.tabi/config/tmp_genesis_step_1.json && rm ~/.tabi/config/tmp_genesis.json
    KEY=$(jq '.body.messages[0].pubkey.key' $FILE -c)
    DELEGATION=$(jq -r '.body.messages[0].value.amount' $FILE)
    POWER=$(($DELEGATION / 1000000))
    jq '.validators['$IDX'] += {"power":"'$POWER'"}' ~/.tabi/config/tmp_genesis_step_1.json > ~/.tabi/config/tmp_genesis_step_2.json && rm ~/.tabi/config/tmp_genesis_step_1.json
    jq '.validators['$IDX'] += {"pub_key":{"type":"tendermint/PubKeyEd25519","value":'$KEY'}}' ~/.tabi/config/tmp_genesis_step_2.json > ~/.tabi/config/tmp_genesis_step_3.json && rm ~/.tabi/config/tmp_genesis_step_2.json
    mv ~/.tabi/config/tmp_genesis_step_3.json ~/.tabi/config/tmp_genesis.json
    IDX=$(($IDX+1))
done

mv ~/.tabi/config/tmp_genesis.json ~/.tabi/config/genesis.json
