package evmrpc

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestCalculatePercentiles(t *testing.T) {
	tests := []struct {
		name              string
		rewardPercentiles []float64
		gasAndRewards     []GasAndReward
		totalGasUsed      uint64
		wantLen           int
	}{
		{
			name:              "empty rewards returns zeros",
			rewardPercentiles: []float64{50},
			gasAndRewards:     []GasAndReward{},
			totalGasUsed:      0,
			wantLen:           1,
		},
		{
			name:              "single tx median",
			rewardPercentiles: []float64{50},
			gasAndRewards: []GasAndReward{
				{GasUsed: 21000, Reward: big.NewInt(1000000000)},
			},
			totalGasUsed: 21000,
			wantLen:      1,
		},
		{
			name:              "multiple percentiles",
			rewardPercentiles: []float64{25, 50, 75},
			gasAndRewards: []GasAndReward{
				{GasUsed: 21000, Reward: big.NewInt(1000000000)},
				{GasUsed: 21000, Reward: big.NewInt(2000000000)},
				{GasUsed: 21000, Reward: big.NewInt(3000000000)},
				{GasUsed: 21000, Reward: big.NewInt(4000000000)},
			},
			totalGasUsed: 84000,
			wantLen:      3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePercentiles(tt.rewardPercentiles, tt.gasAndRewards, tt.totalGasUsed)
			if len(result) != tt.wantLen {
				t.Errorf("CalculatePercentiles() got %d results, want %d", len(result), tt.wantLen)
			}
			for i, r := range result {
				if r == nil {
					t.Errorf("CalculatePercentiles() result[%d] is nil", i)
				}
			}
		})
	}
}

func TestCalculatePercentilesZeroReturnsZeros(t *testing.T) {
	result := CalculatePercentiles([]float64{25, 50, 75}, []GasAndReward{}, 0)
	for i, r := range result {
		if r.ToInt().Cmp(big.NewInt(0)) != 0 {
			t.Errorf("Expected zero at index %d, got %s", i, r.ToInt().String())
		}
	}
}

func TestCalculatePercentilesOrdering(t *testing.T) {
	gasAndRewards := []GasAndReward{
		{GasUsed: 21000, Reward: big.NewInt(3000000000)},
		{GasUsed: 21000, Reward: big.NewInt(1000000000)},
		{GasUsed: 21000, Reward: big.NewInt(2000000000)},
	}
	result := CalculatePercentiles([]float64{33, 66, 100}, gasAndRewards, 63000)

	for i := 1; i < len(result); i++ {
		if result[i].ToInt().Cmp(result[i-1].ToInt()) < 0 {
			t.Errorf("Results should be non-decreasing: result[%d]=%s < result[%d]=%s",
				i, result[i].ToInt().String(), i-1, result[i-1].ToInt().String())
		}
	}
}

func TestCalculatePercentilesExactValues(t *testing.T) {
	gasAndRewards := []GasAndReward{
		{GasUsed: 25000, Reward: big.NewInt(1000000000)},
		{GasUsed: 25000, Reward: big.NewInt(2000000000)},
		{GasUsed: 25000, Reward: big.NewInt(3000000000)},
		{GasUsed: 25000, Reward: big.NewInt(4000000000)},
	}
	totalGas := uint64(100000)

	result := CalculatePercentiles([]float64{25, 50, 75}, gasAndRewards, totalGas)

	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}

	expected := []*big.Int{
		big.NewInt(1000000000),
		big.NewInt(2000000000),
		big.NewInt(3000000000),
	}

	for i, exp := range expected {
		if result[i].ToInt().Cmp(exp) != 0 {
			t.Errorf("Percentile %d: expected %s, got %s",
				i, exp.String(), result[i].ToInt().String())
		}
	}
}

func TestCalculatePercentilesSingleTxReturnsItsReward(t *testing.T) {
	gasAndRewards := []GasAndReward{
		{GasUsed: 21000, Reward: big.NewInt(5000000000)},
	}
	result := CalculatePercentiles([]float64{50}, gasAndRewards, 21000)

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if result[0].ToInt().Cmp(big.NewInt(5000000000)) != 0 {
		t.Errorf("Expected 5000000000, got %s", result[0].ToInt().String())
	}
}

func TestGasAndRewardStruct(t *testing.T) {
	gr := GasAndReward{
		GasUsed: 21000,
		Reward:  big.NewInt(1000000000),
	}
	if gr.GasUsed != 21000 {
		t.Errorf("Expected GasUsed 21000, got %d", gr.GasUsed)
	}
	if gr.Reward.Cmp(big.NewInt(1000000000)) != 0 {
		t.Errorf("Expected Reward 1000000000, got %s", gr.Reward.String())
	}
}

func TestFeeHistoryResultStruct(t *testing.T) {
	result := FeeHistoryResult{
		OldestBlock:  (*hexutil.Big)(big.NewInt(100)),
		BaseFee:      []*hexutil.Big{(*hexutil.Big)(big.NewInt(1000))},
		GasUsedRatio: []float64{0.5},
		Reward:       [][]*hexutil.Big{{(*hexutil.Big)(big.NewInt(500))}},
	}
	if result.OldestBlock.ToInt().Int64() != 100 {
		t.Errorf("Expected OldestBlock 100, got %d", result.OldestBlock.ToInt().Int64())
	}
}
