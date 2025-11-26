package usecase

import (
	"sync"
	"testing"
)

func TestCompile(t *testing.T) {
	uc := NewCompilerUsecase(nil, "/noted/codes/kernels", "noted-kernel_")	

	uc.kernelMuxes["11"] = &sync.Mutex{}
	err := uc.RunBlock("1", "4bcb102d_d663_4bec_86b4_86e978b5b54c", "1")

	if err != nil {
		t.Fatalf("%s", err.Error())
	}
}
	