<template>
  <div class="vm-progress"
       :class="[
         'vm-progress--' + type,
          status ? 'is-' + status : '',
          {
            'vm-progress--without-text': !showText,
            'vm-progress--text-inside': textInside
          }
        ]">
    <div class="vm-progress-bar" v-if="type === 'line'">
      <div class="vm-progress-bar__outer" :style="{ height: strokeWidth + 'px', backgroundColor: trackColor }">
        <div class="vm-progress-bar__inner" :class="[{'vm-progress-bar__striped': striped}, linearClassName]" :style="barStyle">
          <div class="vm-progress-bar__innerText" v-if="showText && textInside"><slot>{{percentage}}%</slot></div>
        </div>
      </div>
    </div>
    <div class="vm-progress-circle" :style="{height: width + 'px', width: width + 'px'}" v-else>
      <svg viewBox="0 0 100 100">
        <path class="vm-progress-circle__track" :d="trackPath" :stroke="trackColor" :stroke-width="relativeStrokeWidth"
              fill="none"></path>
        <path class="vm-progress-circle__path" :d="trackPath" :stroke-linecap="strokeLinecap" :stroke="stroke"
              :stroke-width="relativeStrokeWidth" fill="none" :style="circlePathStyle"></path>
      </svg>
    </div>
    <div class="vm-progress__text"
         v-if="showText && !textInside"
         ref="progressText"
         :style="{ fontSize: progressTextSize + 'px' }">
      <template v-if="!st || strokeColor || $slots.default">
        <slot>{{percentage}}%</slot>
      </template>
      <i v-else :class="iconClass"></i>
    </div>
  </div>
</template>
<script>
  export default {
    name: 'VmProgress',
    componentName: 'VmProgress',
    props: {
      type: {
        type: String,
        default: 'line',
        validator: val => {
          return ['line', 'circle'].indexOf(val) > -1
        }
      },
      percentage: {
        type: [Number, String],
        default: 0,
        required: true,
        validator: val => {
          return val >= 0 && val <= 100
        }
      },
      strokeWidth: {
        type: [Number, String],
        default: 6
      },
      strokeLinecap: {
        type: String,
        default: 'round',
        validator: val => {
          return ['butt', 'square', 'round'].indexOf(val) > -1
        }
      },
      strokeColor: {
        type: String
      },
      trackColor: {
        type: String,
        default () {
          return this.type === 'line' ? '#e4e8f1' : '#e5e9f2'
        }
      },
      textInside: {
        type: Boolean,
        default: false
      },
      showText: {
        type: Boolean,
        default: true
      },
      status: {
        type: String,
        validator: val => {
          return ['success', 'exception', 'warning', 'info'].indexOf(val) > -1
        }
      },
      width: {
        type: Number,
        default: 126
      },
      reverse: {
        type: Boolean,
        default: false
      },
      striped: {
        type: Boolean,
        default: false
      },
      linearClassName: String
    },
    data () {
      return {
        st: this.status
      }
    },
    watch: {
      percentage (newVal) {
        if (this.$slots.default) return
        this.st = newVal === 100 ? 'success' : this.status
      },
      status (newVal) {
        this.st = newVal
      }
    },
    computed: {
      barStyle () {
        let style = {}
        style.width = this.percentage + '%'
        if (this.strokeColor) {
          style.backgroundColor = this.strokeColor
        }
        return style
      },
      relativeStrokeWidth () {
        return (this.strokeWidth / this.width * 100).toFixed(1)
      },
      trackPath () {
        let radius = parseInt(50 - parseFloat(this.relativeStrokeWidth) / 2, 10)
        let reverse = this.reverse ? 0 : 1
        return `M 50 50 m 0 -${radius} a ${radius} ${radius} 0 1 ${reverse} 0 ${radius * 2} a ${radius} ${radius} 0 1 ${reverse} 0 -${radius * 2}`
      },
      perimeter () {
        let radius = 50 - parseFloat(this.relativeStrokeWidth) / 2
        return 2 * Math.PI * radius
      },
      circlePathStyle () {
        let perimeter = this.perimeter;
        return {
          strokeDasharray: `${perimeter}px,${perimeter}px`,
          strokeDashoffset: (1 - this.percentage / 100) * perimeter + 'px',
          transition: 'stroke-dashoffset 0.6s ease 0s, stroke 0.6s ease'
        }
      },
      stroke () {
        let ret
        switch (this.st) {
          case 'success':
            ret = '#13ce66'
            break
          case 'warning':
            ret = '#f7ba2a'
            break
          case 'info':
            ret = '#50bfff'
            break
          case 'exception':
            ret = '#ff4949'
            break
          default:
            ret = this.strokeColor ? this.strokeColor : '#20a0ff'
        }
        return ret
      },
      iconClass () {
        let prefix = `vm-progress-icon${this.type === 'line' ? '-circle' : ''}--`
        return prefix + (this.st === 'exception' ? 'error' : this.st)
      },
      progressTextSize () {
        return this.type === 'line'
          ? 12 + this.strokeWidth * .4
          : this.width * 0.111111 + 2
      }
    }
  }
</script>

<style lang="less">
@css-prefix: ~'vm-';

@color-white: #fff;
@color-success: #13ce66;
@color-warning: #f7ba2a;
@color-danger: #ff4949;
@color-info: #50bfff;

@bg-color: #e4e8f1;
@text-color: #48576a;
@progress-primary-color: #20a0ff;
@progress-success-color: @color-success;
@progress-warning-color: @color-warning;
@progress-info-color: @color-info;
@progress-danger-color: @color-danger;
@progress-white-color: @color-white;

@progress-icon-prefix-cls: ~"@{css-prefix}progress-icon";

@font-face {
  font-family: "iconfont";
  src: url('fonts/iconfont.eot?t=1500451929261'); /* IE9*/
  src: url('fonts/iconfont.eot?t=1500451929261#iefix') format('embedded-opentype'), /* IE6-IE8 */ url('fonts/iconfont.woff?t=1500451929261') format('woff'), /* chrome, firefox */ url('fonts/iconfont.ttf?t=1500451929261') format('truetype'), /* chrome, firefox, opera, Safari, Android, iOS 4.2+*/ url('fonts/iconfont.svg?t=1500451929261#iconfont') format('svg'); /* iOS 4.1- */
}

[class^="@{progress-icon-prefix-cls}"], [class*=" @{progress-icon-prefix-cls}"] {
  font-family: "iconfont" !important;
  font-size: 16px;
  font-style: normal;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

.@{progress-icon-prefix-cls} {
  &-circle--success {
    color: @progress-success-color;
    &:before {
      content: "\e8e4";
    }
  }
  &--success {
    color: @progress-success-color;
    &:before {
      content: "\e8e5";
    }
  }
  &-circle--error {
    color: @progress-danger-color;
    &:before {
      content: "\e8e7";
    }
  }
  &--error {
    color: @progress-danger-color;
    &:before {
      content: "\e8e8";
    }
  }
  &-circle--info {
    color: @progress-info-color;
    &:before {
      content: "\e8ea";
    }
  }
  &--info {
    color: @progress-info-color;
    &:before {
      content: "\e8e9";
    }
  }
  &-circle--warning {
    color: @progress-warning-color;
    &:before {
      content: "\e8ec";
    }
  }
  &--warning {
    color: @progress-warning-color;
    &:before {
      content: "\e8ee";
    }
  }
  &--close {
    &:before {
      content: "\e8e8";
    }
  }
}

.@{css-prefix}progress {
  position: relative;
  line-height: 1;

  &__text {
	display: inline-block;
	vertical-align: middle;
	margin-left: 10px;
	font-size: 14px;
	color: @text-color;
	line-height: 1;
  }

  &--circle {
	display: inline-block;

	.vm-progress__text {
	  position: absolute;
	  top: 50%;
	  left: 0;
	  width: 100%;
	  margin: 0;
	  text-align: center;
	  transform: translate(0, -50%);

	  i {
		display: inline-block;
		vertical-align: middle;
		font-size: 22px;
		font-weight: bold;
	  }
	}
  }

  &.is-success {
	.vm-progress-bar__inner {
	  background-color: @progress-success-color;
	}
	.vm-progress__text {
	  color: @progress-success-color;
	}
  }

  &.is-exception {
	.vm-progress-bar__inner {
	  background-color: @progress-danger-color;
	}
	.vm-progress__text {
	  color: @progress-danger-color;
	}
  }

  &.is-warning {
	.vm-progress-bar__inner {
	  background-color: @progress-warning-color;
	}
	.vm-progress__text {
	  color: @progress-warning-color;
	}
  }

  &.is-info {
	.vm-progress-bar__inner {
	  background-color: @progress-info-color;
	}
	.vm-progress__text {
	  color: @progress-info-color;
	}
  }


  &--without-text {
	.vm-progress__text {
	  display: none;
	}
	.vm-progress-bar {
	  padding-right: 0;
	  margin-right: 0;
	  display: block;
	}
  }

  &--text-inside {
	.vm-progress-bar {
	  padding-right: 0;
	  margin-right: 0;
	}
  }
  &-bar {
	display: inline-block;
	vertical-align: middle;
	width: 100%;
	padding-right: 50px;
	margin-right: -55px;
	box-sizing: border-box;
	&__outer {
	  position: relative;
	  height: 6px;
	  background-color: @bg-color;
	  border-radius: 100px;
	  vertical-align: middle;
	  overflow: hidden;
	}
	&__inner {
	  position: absolute;
	  left: 0;
	  top: 0;
	  height: 100%;
	  line-height: 1;
	  text-align: right;
	  background-color: @progress-primary-color;
	  border-radius: 100px;
	}
	&__innerText {
	  display: inline-block;
	  vertical-align: middle;
	  color: @progress-white-color;
	  font-size: 12px;
	  margin: 0 5px;
	  white-space: nowrap;
	}
	&__striped {
	  background-image: linear-gradient(45deg, rgba(255, 255, 255, .15) 25%, transparent 25%, transparent 50%, rgba(255, 255, 255, .15) 50%, rgba(255, 255, 255, .15) 75%, transparent 75%, transparent);
	  background-size: 40px 40px;
	  animation: progress-bar-stripes 2s linear infinite
	}
  }
}
@keyframes progress-bar-stripes {
  from {
	background-position: 40px 0
  }

  to {
	background-position: 0 0
  }
}
</style>
