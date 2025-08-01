package ui

import (
	"fmt"

	"github.com/miu200521358/bone_baker/pkg/domain"
	"github.com/miu200521358/bone_baker/pkg/usecase"
	"github.com/miu200521358/mlib_go/pkg/domain/mmath"
	"github.com/miu200521358/mlib_go/pkg/interface/controller"
	"github.com/miu200521358/mlib_go/pkg/interface/controller/widget"
	"github.com/miu200521358/walk/pkg/walk"
)

type BakeState struct {
	AddSetButton          *widget.MPushButton  // セット追加ボタン
	ResetSetButton        *widget.MPushButton  // セットリセットボタン
	SaveSetButton         *widget.MPushButton  // セット保存ボタン
	LoadSetButton         *widget.MPushButton  // セット読込ボタン
	NavToolBar            *walk.ToolBar        // セットツールバー
	currentIndex          int                  // 現在のインデックス
	OriginalMotionPicker  *widget.FilePicker   // 元モーション
	OriginalModelPicker   *widget.FilePicker   // 物理焼き込み先モデル
	OutputMotionPicker    *widget.FilePicker   // 出力モーション
	OutputModelPicker     *widget.FilePicker   // 出力モデル
	BakedHistoryIndexEdit *walk.NumberEdit     // 出力モーションインデックスプルダウン
	SaveModelButton       *widget.MPushButton  // モデル保存ボタン
	SaveMotionButton      *widget.MPushButton  // モーション保存ボタン
	Player                *widget.MotionPlayer // モーションプレイヤー
	PhysicsTableView      *walk.TableView      // 物理ボーン表示テーブル
	OutputTableView       *walk.TableView      // 出力定義テーブル
	BakeSets              []*domain.BakeSet    `json:"bake_sets"` // ボーン焼き込みセット

	// Usecase（依存性注入）
	bakeUsecase *usecase.BakeUsecase
}

func NewBakeState(bakeUsecase *usecase.BakeUsecase) *BakeState {
	return &BakeState{
		bakeUsecase:  bakeUsecase,
		BakeSets:     make([]*domain.BakeSet, 0),
		currentIndex: -1,
	}
}

func (ss *BakeState) AddAction() {
	index := ss.NavToolBar.Actions().Len()

	action := ss.newAction(index)
	ss.NavToolBar.Actions().Add(action)
	ss.ChangeCurrentAction(index)
}

func (ss *BakeState) newAction(index int) *walk.Action {
	action := walk.NewAction()
	action.SetCheckable(true)
	action.SetExclusive(true)
	action.SetText(fmt.Sprintf(" No. %d ", index+1))

	action.Triggered().Attach(func() {
		ss.ChangeCurrentAction(index)
	})

	return action
}

func (ss *BakeState) ResetSet() {
	// 一旦全部削除
	for range ss.NavToolBar.Actions().Len() {
		index := ss.NavToolBar.Actions().Len() - 1
		ss.BakeSets[index].Delete()
		ss.NavToolBar.Actions().RemoveAt(index)
	}

	ss.BakeSets = make([]*domain.BakeSet, 0)
	ss.currentIndex = -1

	// 1セット追加
	ss.BakeSets = append(ss.BakeSets, domain.NewPhysicsSet(len(ss.BakeSets)))
	ss.AddAction()
}

func (ss *BakeState) ChangeCurrentAction(index int) {
	// 一旦すべてのチェックを外す
	for i := range ss.NavToolBar.Actions().Len() {
		ss.NavToolBar.Actions().At(i).SetChecked(false)
	}

	// 該当INDEXのみチェックON
	ss.currentIndex = index
	ss.NavToolBar.Actions().At(index).SetChecked(true)

	// 物理焼き込みセットの情報を表示
	ss.OriginalModelPicker.ChangePath(ss.CurrentSet().OriginalModelPath)
	ss.OriginalMotionPicker.ChangePath(ss.CurrentSet().OriginalMotionPath)
	ss.OutputModelPicker.ChangePath(ss.CurrentSet().OutputModelPath)
	ss.OutputMotionPicker.ChangePath(ss.CurrentSet().OutputMotionPath)

	// // 物理ツリーのモデル変更
	// if ss.CurrentSet().PhysicsBoneTreeModel == nil {
	// 	ss.CurrentSet().PhysicsBoneTreeModel = domain.NewPhysicsBoneTreeModel()
	// }
	// ss.PhysicsTreeView.SetModel(ss.CurrentSet().PhysicsBoneTreeModel)
}

func (ss *BakeState) ClearOptions() {
	ss.Player.Reset(ss.MaxFrame())
}

func (ss *BakeState) MaxFrame() float32 {
	maxFrame := float32(0)
	for _, physicsSet := range ss.BakeSets {
		if physicsSet.OriginalMotion != nil && maxFrame < physicsSet.OriginalMotion.MaxFrame() {
			maxFrame = physicsSet.OriginalMotion.MaxFrame()
		}
	}

	return maxFrame
}

func (ss *BakeState) SetCurrentIndex(index int) {
	ss.currentIndex = index
}

func (ss *BakeState) CurrentIndex() int {
	return ss.currentIndex
}

func (ss *BakeState) CurrentSet() *domain.BakeSet {
	if ss.currentIndex < 0 || ss.currentIndex >= len(ss.BakeSets) {
		return nil
	}

	return ss.BakeSets[ss.currentIndex]
}

// SaveSet セット情報を保存
func (ss *BakeState) SaveSet(jsonPath string) error {
	return ss.bakeUsecase.SaveBakeSet(ss.BakeSets, jsonPath)
}

// LoadSet セット情報を読み込む
func (ss *BakeState) LoadSet(jsonPath string) error {
	bakeSets, err := ss.bakeUsecase.LoadBakeSet(jsonPath)
	if err != nil {
		return err
	}

	ss.BakeSets = bakeSets
	return nil
}

// LoadModel 元モデルを読み込む
func (bakeState *BakeState) LoadModel(
	cw *controller.ControlWindow, path string,
) error {
	bakeState.SetWidgetEnabled(false)

	// オプションクリア
	bakeState.ClearOptions()

	if err := bakeState.bakeUsecase.LoadModel(bakeState.CurrentSet(), path); err != nil {
		return err
	}

	cw.StoreModel(0, bakeState.CurrentIndex(), bakeState.CurrentSet().OriginalModel)
	cw.StoreModel(1, bakeState.CurrentIndex(), bakeState.CurrentSet().BakedModel)

	cw.StoreMotion(0, bakeState.CurrentIndex(), bakeState.CurrentSet().OriginalMotion)
	if bakeState.CurrentSet().OriginalMotion != nil {
		if copiedMotion, err := bakeState.CurrentSet().OriginalMotion.Copy(); err == nil {
			cw.StoreMotion(1, bakeState.CurrentIndex(), copiedMotion)
		}
	}

	for n := range bakeState.BakeSets {
		cw.ClearDeltaMotion(0, n)
		cw.ClearDeltaMotion(1, n)
		cw.SetSaveDeltaIndex(0, 0)
		cw.SetSaveDeltaIndex(1, 0)
	}

	bakeState.BakedHistoryIndexEdit.SetValue(1.0)
	bakeState.BakedHistoryIndexEdit.SetRange(1.0, 2.0)

	bakeState.OutputModelPicker.ChangePath(bakeState.CurrentSet().OutputModelPath)
	bakeState.SetWidgetEnabled(true)

	return nil
}

// LoadMotion 物理焼き込みモーションを読み込む
func (bakeState *BakeState) LoadMotion(
	cw *controller.ControlWindow, path string, isClear bool,
) error {
	bakeState.SetWidgetEnabled(false)

	// オプションクリア
	if isClear {
		bakeState.ClearOptions()
	}

	if err := bakeState.bakeUsecase.LoadMotion(bakeState.CurrentSet(), path); err != nil {
		return err
	}

	if bakeState.CurrentSet().OriginalMotion != nil {
		cw.StoreMotion(0, bakeState.CurrentIndex(), bakeState.CurrentSet().OriginalMotion)
	}

	if bakeState.CurrentSet().OutputMotion != nil {
		cw.StoreMotion(1, bakeState.CurrentIndex(), bakeState.CurrentSet().OutputMotion)
	}

	for n := range bakeState.BakeSets {
		cw.ClearDeltaMotion(0, n)
		cw.ClearDeltaMotion(1, n)
		cw.SetSaveDeltaIndex(0, 0)
		cw.SetSaveDeltaIndex(1, 0)
	}

	bakeState.BakedHistoryIndexEdit.SetValue(1.0)
	bakeState.BakedHistoryIndexEdit.SetRange(1.0, 2.0)

	if bakeState.CurrentSet().OriginalMotion != nil {
		// 出力ボーン定義に行追加
		bakeState.CurrentSet().OutputTableModel = domain.NewOutputTableModel()
		bakeState.CurrentSet().OutputTableModel.AddRecord(
			bakeState.CurrentSet().OriginalModel,
			0,
			bakeState.CurrentSet().OriginalMotion.MaxFrame())
		bakeState.OutputTableView.SetModel(bakeState.CurrentSet().OutputTableModel)

		// 物理定義に行追加
		bakeState.CurrentSet().PhysicsTableModel = domain.NewPhysicsTableModel()
		bakeState.CurrentSet().PhysicsTableModel.AddRecord(
			bakeState.CurrentSet().OriginalModel,
			0,
			bakeState.CurrentSet().OriginalMotion.MaxFrame())
		bakeState.PhysicsTableView.SetModel(bakeState.CurrentSet().PhysicsTableModel)
	}

	{
		maxFrames := make([]float32, len(bakeState.BakeSets))
		for i, set := range bakeState.BakeSets {
			if set.OriginalMotion != nil {
				maxFrames[i] = set.OriginalMotion.MaxFrame()
			}
		}
		// モーションプレイヤーのリセット
		bakeState.Player.Reset(mmath.Max(maxFrames))
	}

	bakeState.OutputMotionPicker.SetPath(bakeState.CurrentSet().OutputMotionPath)
	bakeState.SetWidgetEnabled(true)

	return nil
}

// SetWidgetEnabled 物理焼き込み有効無効設定
func (bakeState *BakeState) SetWidgetEnabled(enabled bool) {
	bakeState.OutputTableView.SetEnabled(enabled)

	bakeState.AddSetButton.SetEnabled(enabled)
	bakeState.ResetSetButton.SetEnabled(enabled)
	bakeState.SaveSetButton.SetEnabled(enabled)
	bakeState.LoadSetButton.SetEnabled(enabled)

	bakeState.OriginalMotionPicker.SetEnabled(enabled)
	bakeState.OriginalModelPicker.SetEnabled(enabled)
	bakeState.OutputMotionPicker.SetEnabled(enabled)
	bakeState.OutputModelPicker.SetEnabled(enabled)

	bakeState.SaveModelButton.SetEnabled(enabled)
	bakeState.SaveMotionButton.SetEnabled(enabled)

	bakeState.SetWidgetPlayingEnabled(enabled)
}

func (bakeState *BakeState) SetWidgetPlayingEnabled(enabled bool) {
	bakeState.Player.SetEnabled(enabled)

	bakeState.PhysicsTableView.SetEnabled(enabled)

	// bakeState.GravityEdit.SetEnabled(enabled)
	// bakeState.MaxSubStepsEdit.SetEnabled(enabled)
	// bakeState.FixedTimeStepEdit.SetEnabled(enabled)
	// bakeState.MassEdit.SetEnabled(enabled)
	// bakeState.StiffnessEdit.SetEnabled(enabled)
	// bakeState.TensionEdit.SetEnabled(enabled)
	// bakeState.PhysicsTreeView.SetEnabled(enabled)
}
