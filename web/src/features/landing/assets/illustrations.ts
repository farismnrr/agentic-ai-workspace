import type { IllustrationAsset } from '../../../shared/ui/Illustration.svelte';
import deskLamp from './desk-lamp-book-coffee.webp';
import deskLaptop from './desk-laptop-jacket.webp';
import deskLaptopSm from './desk-laptop-jacket-sm.webp';
import key from './key.webp';
import openDoor from './open-door.webp';
import openDoorSm from './open-door-sm.webp';
import openSign from './open-sign.webp';
import paperAirplane from './paper-airplane.webp';
import papersPencil from './papers-pencil.webp';
import receipt from './receipt-trust-me.webp';
import rubberStamp from './rubber-stamp.webp';
import rubberStampSm from './rubber-stamp-sm.webp';
import smartphone from './smartphone.webp';
import stampProven from './stamp-proven-to-work.webp';

export const illustrations = {
  deskLamp: { src: deskLamp, width: 1005, height: 754, alt: '' },
  deskLaptop: { src: deskLaptop, width: 945, height: 777, alt: '', srcset: `${deskLaptopSm} 480w, ${deskLaptop} 945w` },
  key: { src: key, width: 603, height: 472, alt: '' },
  openDoor: { src: openDoor, width: 737, height: 739, alt: '', srcset: `${openDoorSm} 480w, ${openDoor} 737w` },
  openSign: { src: openSign, width: 577, height: 515, alt: '' },
  paperAirplane: { src: paperAirplane, width: 971, height: 423, alt: '' },
  papersPencil: { src: papersPencil, width: 697, height: 535, alt: '' },
  receipt: {
    src: receipt,
    width: 965,
    height: 712,
    alt: 'A vague "Trust me, it works now" speech bubble next to a Masih Awam receipt listing the task, what changed, what was tested, and the result: Working.',
  },
  rubberStamp: { src: rubberStamp, width: 666, height: 751, alt: '', srcset: `${rubberStampSm} 480w, ${rubberStamp} 666w` },
  smartphone: { src: smartphone, width: 855, height: 548, alt: '' },
  stampProven: { src: stampProven, width: 839, height: 313, alt: 'Proven to work stamp' },
} satisfies Record<string, IllustrationAsset>;
